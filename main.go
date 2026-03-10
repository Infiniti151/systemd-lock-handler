package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"flag"
	"time"

	"github.com/coreos/go-systemd/v22/daemon"
	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/login1"
	dbus "github.com/godbus/dbus/v5"
)

var lastTriggerTime time.Time
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

func StartSystemdUserUnit(unitName string) error {
    conn, err := systemd.NewUserConnectionContext(context.Background())
    if err != nil {
        return fmt.Errorf("failed to connect to systemd user session: %v", err)
    }
    defer conn.Close()

    ch := make(chan string, 1)

    _, err = conn.StartUnitContext(context.Background(), unitName, "replace", ch)
    if err != nil {
        return fmt.Errorf("failed to start unit: %v", err)
    }

    result := <-ch
    if result == "done" {
        info("Started systemd unit: %s", unitName)
        return nil
    } 
    
    return fmt.Errorf("failed to start unit %s: %v", unitName, result)
}

func ListenForSleep() {
    conn, err := dbus.ConnectSystemBus()
    if err != nil {
        fatal("Could not connect to the system D-Bus: %v", err)
    }

    err = conn.AddMatchSignal(
        dbus.WithMatchObjectPath("/org/freedesktop/login1"),
        dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
        dbus.WithMatchMember("PrepareForSleep"),
    )
    if err != nil {
        fatal("Failed to listen for sleep signals: %v", err)
    }

    c := make(chan *dbus.Signal, 10)
    logind, err := login1.New()
    if err != nil {
        fatal("Failed to connect to logind: %v", err)
    }

    go func() {
        for {
            // We need to inhibit sleeping so we have time to execute our actions before the system sleeps.
            lock, err := logind.Inhibit("sleep", "systemd-lock-handler", "Start pre-sleep target", "delay")
            if err != nil {
                fatal("Failed to grab sleep inhibitor lock: %v", err)
            }
            info("Got lock on sleep inhibitor")

            if err := waitPrepareForSleep(c, true); err != nil {
                fatal("Before releasing inhibitor lock: %v", err)
            }

            info("Action: Triggering sleep.target")

            if err = SafeTrigger("sleep.target"); err != nil {
                errorLog("Error starting sleep.target: %v", err)
            }
            
            // Uninhibit sleeping. I.e.: let the system actually go to sleep.
            if err := lock.Close(); err != nil {
                fatal("Error releasing inhibitor lock: %v", err)
            }

            if err := waitPrepareForSleep(c, false); err != nil {
                fatal("After releasing inhibitor lock: %v", err)
            }

            info("The system is now proceeding to sleep")
        }
    }()

    conn.Signal(c)
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

func StartUnifiedMonitor(u *user.User, sessionPath dbus.ObjectPath, doLock bool, doUnlock bool) {
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

				lastLocked = isLocked

				if isLocked && doLock {
					info("Action: Detected Lock (LockedHint=true)")
					if err := SafeTrigger("lock.target"); err != nil {
						errorLog("Monitor: %v", err)
					}
				} else if !isLocked && doUnlock {
					info("Action: Detected Unlock (LockedHint=false)")
					if err := SafeTrigger("unlock.target"); err != nil {
						errorLog("Monitor: %v", err)
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
    handleSleep := flag.Bool("sleep", true, "Enable detection of PrepareForSleep signals")
    handleLock := flag.Bool("lock", true, "Enable detection of Lock signals")
    handleUnlock := flag.Bool("unlock", true, "Enable detection of Unlock signals")

    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "Listens for logind events to trigger systemd user targets.\n\n")
        fmt.Fprintln(os.Stderr, "Options:")
        flag.PrintDefaults()
    }
    flag.Parse()

    configs := []struct {
        name    string
        enabled bool
    }{
        {"Sleep detection", *handleSleep},
        {"Lock detection", *handleLock},
        {"Unlock detection", *handleUnlock},
    }

    for _, cfg := range configs {
        if !cfg.enabled {
            warn("%s is disabled by flag.", cfg.name)
        } else {
            info("%s is enabled.", cfg.name)
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

    if *handleSleep {
        ListenForSleep()
    }

    if *handleLock || *handleUnlock {
        StartUnifiedMonitor(currUser, sessionPath, *handleLock, *handleUnlock)
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
