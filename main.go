package main

import (
	"context"
	"fmt"
	"log"
	"os/user"
	"strings"

	"github.com/coreos/go-systemd/v22/daemon"
	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/login1"
	dbus "github.com/godbus/dbus/v5"
)

// Starts a systemd unit and blocks until the job is completed.
func StartSystemdUserUnit(unitName string) error {
	conn, err := systemd.NewUserConnectionContext(context.Background())
	if err != nil {
		return fmt.Errorf("failed to connect to systemd user session: %v", err)
	}

	ch := make(chan string, 1)

	_, err = conn.StartUnitContext(context.Background(), unitName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to start unit: %v", err)
	}

	result := <-ch
	if result == "done" {
		log.Println("Started systemd unit:", unitName)
	} else {
		return fmt.Errorf("failed to start unit %v: %v", unitName, result)
	}

	return nil
}

func ListenForSleep() {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatalln("Could not connect to the system D-Bus", err)
	}

	// TODO: Should I also stop `sleep.target` after the system comes back
	// from sleeping? (`lock.target` will continue running anyway).
	err = conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/login1"),
		dbus.WithMatchInterface("org.freedesktop.login1.Manager"),
		dbus.WithMatchMember("PrepareForSleep"),
	)
	if err != nil {
		log.Fatalln("Failed to listen for sleep signals", err)
	}

	c := make(chan *dbus.Signal, 10)
	logind, err := login1.New()
	if err != nil {
		log.Fatalln("Failed to connect to logind")
	}

	go func() {
		for {
			// We need to inhibit sleeping so we have time to execute our actions before the system sleeps.
			lock, err := logind.Inhibit("sleep", "systemd-lock-handler", "Start pre-sleep target", "delay")
			if err != nil {
				log.Fatalln("Failed to grab sleep inhibitor lock", err)
			}
			log.Println("Got lock on sleep inhibitor")

			if err := waitPrepareForSleep(c, true); err != nil {
				log.Fatalln("Before releasing inhibitor lock:", err)
			}

			log.Println("Starting sleep.target")

			if err = StartSystemdUserUnit("sleep.target"); err != nil {
				log.Println("Error starting sleep.target:", err)
			}
			// Uninhibit sleeping. I.e.: let the system actually go to sleep.
			if err := lock.Close(); err != nil {
				log.Fatalln("Error releasing inhibitor lock:", err)
			}

			if err := waitPrepareForSleep(c, false); err != nil {
				log.Fatalln("After releasing inhibitor lock:", err)
			}

			log.Println("Started sleep.target, the system is going to sleep")
		}
	}()

	conn.Signal(c)
	log.Println("Listening for sleep events...")
}

func waitPrepareForSleep(c <-chan *dbus.Signal, want bool) error {
	s := <-c

	if len(s.Body) == 0 {
		return fmt.Errorf("empty signal arguments: %v", s)
	}

	got, ok := s.Body[0].(bool)
	if !ok {
		return fmt.Errorf("active argument not a bool: %v", s.Body[0])
	}

	if want != got {
		return fmt.Errorf("expected PrepareForSleep(%v), got %v", want, got)
	}

	return nil
}

func ListenForLock(user *user.User) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatalln("Could not connect to the system D-Bus", err)
	}

	err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.login1.Session"),
		dbus.WithMatchSender("org.freedesktop.login1"),
		dbus.WithMatchMember("Lock"),
	)
	if err != nil {
		log.Fatalln("Failed to listen for lock signals", err)
	}

	err = conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.login1.Session"),
		dbus.WithMatchSender("org.freedesktop.login1"),
		dbus.WithMatchMember("Unlock"),
	)
	if err != nil {
		log.Fatalln("Failed to listen for unlock signals", err)
	}

	c := make(chan *dbus.Signal, 10)
	go func() {
		for {
			v := <-c

			var target string
			signalName := v.Name
			if strings.HasSuffix(signalName, "Lock") {
				target = "lock.target"
			} else if strings.HasSuffix(signalName, "Unlock") {
				target = "unlock.target"
			} else {
				log.Println("Got an unknown event!", v)
				continue
			}

			// Get the (un)locked session...
			obj := conn.Object("org.freedesktop.login1", v.Path)

			name, err := obj.GetProperty("org.freedesktop.login1.Session.Name")
			if err != nil {
				log.Println("WARNING: Could not obtain details for locked session:", err)
				continue
			}

			// ... And check that it belongs to the current user:
			if name.Value() != user.Username {
				continue
			}
			log.Println("Session signal for current user:", signalName)

			if err = StartSystemdUserUnit(target); err != nil {
				log.Println("Error starting target:", target, err)
			}
		}
	}()

	conn.Signal(c)
	log.Println("Listening for lock events...")
}

func main() {
	log.SetFlags(log.Lshortfile)

	user, err := user.Current()
	if err != nil {
		log.Fatalln("Failed to get username:", err)
	}
	log.Println("Running for user:", user.Username)

	//ListenForSleep()
	ListenForLock(user)

	log.Println("Initialization complete.")

	sent, err := daemon.SdNotify(true, daemon.SdNotifyReady)
	if !sent {
		log.Println("Couldn't call sd_notify. Not running via systemd?")
	}
	if err != nil {
		log.Println("Call to sd_notify failed:", err)
	}

	select {}
}
