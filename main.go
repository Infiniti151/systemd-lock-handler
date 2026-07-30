package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"sync/atomic"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/login1"
	dbus "github.com/godbus/dbus/v5"
)

var lastTriggerTime time.Time
var isSleeping atomic.Int32
var resumeCooldown = time.Now()
var allTargets = []string{"lock.target", "unlock.target", "sleep.target", "wake.target"}

const cooldown = 2 * time.Second

func info(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	fmt.Fprintf(os.Stdout, "<6>[OK] %s\n", msg)
}

func warn(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	fmt.Fprintf(os.Stdout, "<4>[!] %s\n", msg)
}

func errorLog(message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	fmt.Fprintf(os.Stderr, "<3>[ERROR] %s\n", msg)
}

func fatal(message string, args ...interface{}) {
	errorLog(message, args...)
	os.Exit(1)
}

func SafeTrigger(target string) error {
	if !lastTriggerTime.IsZero() && time.Since(lastTriggerTime) < cooldown {
		info("Debouncing: %s suppressed (too soon after last event)", target)
		return nil
	}

	lastTriggerTime = time.Now()
	return StartSystemdUserUnit(target)
}

func TriggerExclusive(targetToStart string) error {
	info("Exclusive Trigger: Starting %s and stopping others", targetToStart)

	// Stop all others
	for _, t := range allTargets {
		if t != targetToStart {
			go func(unit string) {
				if err := StopSystemdUserUnit(unit); err != nil {
					errorLog("Failed to stop %s: %v", unit, err)
				}
			}(t)
		}
	}

	return StartSystemdUserUnit(targetToStart)
}

func StartSystemdUserUnit(unitName string) error {
	conn, err := systemd.NewUserConnectionContext(context.Background())
	if err != nil {
		return fmt.Errorf("D-Bus connection failed: %v", err)
	}
	defer conn.Close()

	ch := make(chan string, 1)

	_, err = conn.StartUnitContext(context.Background(), unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to start %s: %v", unitName, err)
	}

	result := <-ch
	if result == "done" {
		info("Started systemd unit: %s", unitName)
		return nil
	}

	return fmt.Errorf("failed to start unit %s: %v", unitName, result)
}

func StopSystemdUserUnit(unitName string) error {
	conn, err := systemd.NewUserConnectionContext(context.Background())
	if err != nil {
		return fmt.Errorf("D-Bus connection failed: %v", err)
	}
	defer conn.Close()

	ch := make(chan string, 1)
	_, err = conn.StopUnitContext(context.Background(), unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to stop %s: %v", unitName, err)
	}

	result := <-ch
	if result == "done" {
		return nil
	}
	return fmt.Errorf("failed to stop unit %s: %v", unitName, result)
}

func ListenForSleep(triggerSleep bool, triggerWake bool) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		fatal("Could not connect to the system D-Bus: %v", err)
	}

	// Match signal for PrepareForSleep
	err = conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/login1"),
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
		dbus.WithMatchMember("PrepareForSleep"),
	)
	if err != nil {
		fatal("Failed to listen for sleep signals: %v", err)
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	logind, _ := login1.New()

	go func() {
		for {
			lock, _ := logind.Inhibit("sleep", "systemd-lock-handler", "Filter lock events", "delay")

			// SLEEP PHASE
			if err := waitPrepareForSleep(c, true); err != nil {
				return
			}

			if triggerSleep {
				isSleeping.Store(1)
				info("Action: Suspending")
				TriggerExclusive("sleep.target")
			}

			lock.Close()

			// WAKE PHASE
			if err := waitPrepareForSleep(c, false); err != nil {
				return
			}

			if triggerWake {
				info("Action: Resuming")
				resumeCooldown = time.Now()
				if err := TriggerExclusive("wake.target"); err != nil {
					errorLog("Sleep Handler: Failed to trigger exclusive wake: %v", err)
				}
			}

			isSleeping.Store(0)
			info("Internal state: READY")
		}
	}()
}

func waitPrepareForSleep(c <-chan *dbus.Signal, want bool) error {
	for s := range c {
		if len(s.Body) == 0 {
			return fmt.Errorf("empty signal arguments: %v", s)
		}

		got, ok := s.Body[0].(bool)
		if !ok {
			return fmt.Errorf("active argument not a bool: %v", s.Body[0])
		}

		if got == want {
			return nil
		}

		warn("Received PrepareForSleep(%v) but waiting for %v. Skipping...", got, want)
	}
	return fmt.Errorf("signal channel closed")
}

func StartUnifiedMonitor(sessionPath dbus.ObjectPath, doLock bool, doUnlock bool) {
	go func() {
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			errorLog("Session Monitor: D-Bus Connect Error: %v", err)
			return
		}
		defer conn.Close()

		err = conn.AddMatchSignal(
			dbus.WithMatchObjectPath(sessionPath),
			dbus.WithMatchInterface("org.freedesktop.DBus.Properties"),
			dbus.WithMatchMember("PropertiesChanged"),
		)
		if err != nil {
			errorLog("Session Monitor: Failed to add match rule: %v", err)
			return
		}

		c := make(chan *dbus.Signal, 10)
		conn.Signal(c)

		var lastLocked bool
		variant, err := conn.Object("org.freedesktop.login1", sessionPath).GetProperty("org.freedesktop.login1.Session.LockedHint")
		if err == nil {
			lastLocked, _ = variant.Value().(bool)
		}

		info("Session Monitor: Watching LockedHint for %s", sessionPath)

		for v := range c {
			if len(v.Body) < 2 {
				continue
			}
			props, ok := v.Body[1].(map[string]dbus.Variant)
			if !ok {
				continue
			}

			if variant, exists := props["LockedHint"]; exists {
				isLocked, ok := variant.Value().(bool)

				if !ok || isLocked == lastLocked {
					continue
				}

				if isSleeping.Load() == 1 {
					info("Monitor: Ignoring LockedHint (System state is BUSY)")
					continue
				}

				if time.Since(resumeCooldown) < 3*time.Second {
					info("Monitor: Ignoring LockedHint (Within resume cooldown)")
					continue
				}

				lastLocked = isLocked

				if isLocked && doLock {
					info("Action: Locking")
					if err := TriggerExclusive("lock.target"); err != nil {
						errorLog("Monitor: Failed to lock: %v", err)
					}
				} else if !isLocked && doUnlock {
					info("Action: Unlocking")
					if err := TriggerExclusive("unlock.target"); err != nil {
						errorLog("Monitor: Failed to unlock: %v", err)
					}
				}
			}
		}
	}()
}

func GetActiveGraphicalSession(u *user.User) dbus.ObjectPath {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		errorLog("Session Finder: D-Bus Connect Error: %v", err)
		return ""
	}
	defer conn.Close()

	var sessions [][]interface{}
	err = conn.Object("org.freedesktop.login1", "/org/freedesktop/login1").Call(
		"org.freedesktop.login1.Manager.ListSessions", 0,
	).Store(&sessions)

	if err != nil {
		errorLog("Session Finder: ListSessions call failed: %v", err)
		return ""
	}

	for _, s := range sessions {
		if len(s) < 5 {
			continue
		}

		username, _ := s[2].(string)
		path, _ := s[4].(dbus.ObjectPath)

		if username == u.Username {
			obj := conn.Object("org.freedesktop.login1", path)

			vType, err := obj.GetProperty("org.freedesktop.login1.Session.Type")
			if err != nil {
				continue
			}

			vActive, err := obj.GetProperty("org.freedesktop.login1.Session.Active")
			if err != nil {
				continue
			}

			sType, _ := vType.Value().(string)
			sActive, _ := vActive.Value().(bool)

			if (sType == "wayland" || sType == "x11") && sActive {
				info("Session Finder: Found active %s session for %s", sType, username)
				return path
			}
		}
	}

	warn("Session Finder: No active graphical session found for user %s", u.Username)
	return ""
}

func main() {
	handleSleep := flag.Bool("sleep", true, "User-level suspend detection (sleep.target)")
	handleWake := flag.Bool("wake", true, "User-level resume detection (wake.target)")
	handleLock := flag.Bool("lock", true, "Lock detection (lock.target)")
	handleUnlock := flag.Bool("unlock", true, "Unlock detection (unlock.target)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Listens for logind events to trigger systemd user targets.\n\n")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	configs := []struct {
		name         string
		current      bool
		defaultValue bool
	}{
		{"Sleep detection", *handleSleep, true},
		{"Wake detection", *handleWake, true},
		{"Lock detection", *handleLock, true},
		{"Unlock detection", *handleUnlock, true},
	}

	for _, cfg := range configs {
		status := "disabled"
		if cfg.current {
			status = "enabled"
		}

		if cfg.current != cfg.defaultValue {
			warn("%s is %s (Non-default).", cfg.name, status)
		} else {
			info("%s is %s.", cfg.name, status)
		}
	}

	currUser, err := user.Current()
	if err != nil {
		fatal("Failed to get username: %v", err)
	}

	sessionPath := GetActiveGraphicalSession(currUser)
	if sessionPath == "" {
		fatal("No active graphical session found for user %s", currUser.Username)
	}

	if *handleSleep || *handleWake {
		ListenForSleep(*handleSleep, *handleWake)
	}

	if *handleLock || *handleUnlock {
		StartUnifiedMonitor(sessionPath, *handleLock, *handleUnlock)
	}

	info("Initialization complete. Running for user: %s", currUser.Username)

	sent, err := daemon.SdNotify(true, daemon.SdNotifyReady)
	if err != nil {
		errorLog("Error calling sd_notify: %v", err)
	} else if !sent {
		warn("Note: sd_notify not sent (likely not running under systemd Type=notify)")
	}

	select {}
}
