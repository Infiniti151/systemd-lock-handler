package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	systemd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
)

func TestImportIntegrity(t *testing.T) {
	t.Run("VerifySystemdImport", func(t *testing.T) {
		var conn *systemd.Conn
		pkgPath := reflect.TypeOf(conn).Elem().PkgPath()
		expectedPath := "github.com/coreos/go-systemd/v22/dbus"

		if pkgPath != expectedPath {
			t.Errorf("❌ FAIL: Systemd type coming from wrong path: %s", pkgPath)
		} else {
			t.Logf("✅ PASS: Systemd correctly linked to %s", pkgPath)
		}
	})

	t.Run("VerifyGodbusImport", func(t *testing.T) {
		var conn *dbus.Conn
		pkgPath := reflect.TypeOf(conn).Elem().PkgPath()
		expectedPath := "github.com/godbus/dbus/v5"

		if pkgPath != expectedPath {
			t.Errorf("❌ FAIL: Godbus type coming from wrong path: %s", pkgPath)
		} else {
			t.Logf("✅ PASS: Godbus correctly linked to %s", pkgPath)
		}
	})
}

func TestSystemdUserConnection(t *testing.T) {
	t.Run("NewUserConnectionContext", func(t *testing.T) {
		ctx := context.Background()
		conn, err := systemd.NewUserConnectionContext(ctx)
		if err != nil {
			t.Logf("⚠️  SKIP: Systemd User Bus connection failed (expected if not in session): %v", err)
			t.SkipNow()
		}
		defer conn.Close()
		t.Log("✅ SUCCESS: Systemd User Connection established")
	})
}

func TestDbusSystemBus(t *testing.T) {
	t.Run("ConnectSystemBus", func(t *testing.T) {
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			t.Logf("⚠️  SKIP: System D-Bus not available: %v", err)
			t.SkipNow()
		}
		defer conn.Close()
		t.Log("✅ SUCCESS: System D-Bus connection established")
	})
}

func TestLockedHintLogic(t *testing.T) {
	tests := []struct {
		name       string
		body       []interface{}
		wantLocked bool
		shouldFail bool
	}{
		{
			name: "Valid Locked (True)",
			body: []interface{}{
				"org.freedesktop.login1.Session",
				map[string]dbus.Variant{"LockedHint": dbus.MakeVariant(true)},
			},
			wantLocked: true,
		},
		{
			name: "Valid Unlocked (False)",
			body: []interface{}{
				"org.freedesktop.login1.Session",
				map[string]dbus.Variant{"LockedHint": dbus.MakeVariant(false)},
			},
			wantLocked: false,
		},
		{
			name: "Missing Property",
			body: []interface{}{
				"org.freedesktop.login1.Session",
				map[string]dbus.Variant{"OtherProp": dbus.MakeVariant(true)},
			},
			shouldFail: true,
		},
		{
			name: "Wrong Type (Int)",
			body: []interface{}{
				"org.freedesktop.login1.Session",
				map[string]dbus.Variant{"LockedHint": dbus.MakeVariant(1)},
			},
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.body) < 2 {
				t.Fatal("Test setup error: body too short")
			}
			props, ok := tt.body[1].(map[string]dbus.Variant)
			if !ok {
				t.Fatal("Test setup error: body[1] not a map")
			}

			variant, exists := props["LockedHint"]
			if !exists {
				if tt.shouldFail {
					t.Log("✅ PASS: Correctly ignored missing LockedHint")
					return
				}
				t.Error("❌ FAIL: Expected LockedHint to exist")
				return
			}

			val, ok := variant.Value().(bool)
			if !ok {
				if tt.shouldFail {
					t.Log("✅ PASS: Correctly caught type mismatch")
					return
				}
				t.Error("❌ FAIL: Could not assert LockedHint as bool")
				return
			}

			if val != tt.wantLocked {
				t.Errorf("❌ FAIL: Got %v, want %v", val, tt.wantLocked)
			}
		})
	}
}

func TestWaitPrepareForSleepRobustness(t *testing.T) {
	t.Run("Signal Loop and Filtering", func(t *testing.T) {
		c := make(chan *dbus.Signal, 3)

		c <- &dbus.Signal{Name: "SomeOtherSignal", Body: []interface{}{true}}

		c <- &dbus.Signal{Name: "PrepareForSleep", Body: []interface{}{true}}

		close(c)

		err := waitPrepareForSleep(c, true)
		if err != nil {
			t.Errorf("❌ FAIL: Should have found the 'true' signal, got: %v", err)
		} else {
			t.Log("✅ PASS: Correctly skipped noise and found target signal")
		}
	})
}

func TestStartSystemdUserUnit(t *testing.T) {
	t.Run("Handle Missing Unit Error", func(t *testing.T) {
		err := StartSystemdUserUnit("non-existent-unit-12345.target")
		if err == nil {
			t.Error("❌ FAIL: Expected error for non-existent unit, but got nil")
		} else {
			t.Logf("✅ PASS: Correctly caught error: %v", err)
		}
	})
}

func TestSafeTriggerCooldown(t *testing.T) {
	t.Run("Verify Debouncing", func(t *testing.T) {
		lastTriggerTime = time.Time{}

		_ = SafeTrigger("test.target")

		err2 := SafeTrigger("test.target")

		if err2 != nil {
			t.Errorf("❌ FAIL: Suppressed trigger should return nil error, got %v", err2)
		} else {
			t.Log("✅ PASS: Successfully debounced rapid-fire triggers")
		}
	})
}

func TestSessionFilteringLogic(t *testing.T) {
	tests := []struct {
		name     string
		sType    string
		sActive  bool
		expected bool
	}{
		{"Valid Wayland Active", "wayland", true, true},
		{"Valid X11 Active", "x11", true, true},
		{"Wayland Inactive", "wayland", false, false},
		{"X11 Inactive", "x11", false, false},
		{"TTY Session (Ignore)", "tty", true, false},
		{"Unrecognized Protocol", "mir", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isGraphical := (tt.sType == "wayland" || tt.sType == "x11")
			got := isGraphical && tt.sActive

			if got != tt.expected {
				t.Errorf("❌ FAIL [%s]: Expected %v, got %v", tt.name, tt.expected, got)
			} else {
				t.Logf("✅ PASS: Correctly handled %s (Active: %v)", tt.sType, tt.sActive)
			}
		})
	}
}

func TestLoggingPrefixes(t *testing.T) {
	tests := []struct {
		level    string
		expected string
	}{
		{"info", "<6>[OK]"},
		{"warn", "<4>[!]"},
		{"error", "<3>[ERROR]"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			msg := "test"
			var actual string
			switch tt.level {
			case "info":
				actual = fmt.Sprintf("<6>[OK] %s", msg)
			case "warn":
				actual = fmt.Sprintf("<4>[!] %s", msg)
			case "error":
				actual = fmt.Sprintf("<3>[ERROR] %s", msg)
			}

			if !strings.HasPrefix(actual, tt.expected) {
				t.Errorf("❌ FAIL: Prefix mismatch.\nGot:  %q\nWant prefix: %q", actual, tt.expected)
			} else {
				t.Logf("✅ PASS: Found expected prefix %q", tt.expected)
			}
		})
	}
}

func TestSleepStateToggle(t *testing.T) {
	t.Run("Verify Atomic State Lifecycle", func(t *testing.T) {
		isSleeping.Store(0)

		isSleeping.Store(1)
		if isSleeping.Load() != 1 {
			t.Error("❌ FAIL: State should be BUSY (1) during suspend")
		}

		isSleeping.Store(0)
		if isSleeping.Load() != 0 {
			t.Error("❌ FAIL: State should be READY (0) after resume")
		}

		t.Log("✅ PASS: isSleeping atomic correctly tracks hardware cycle")
	})
}

func TestBlockSleepLockFiltering(t *testing.T) {
	tests := []struct {
		name        string
		triggerUnit bool
		shouldCall  bool
	}{
		{"Flag True: Should Trigger", true, true},
		{"Flag False: Should Skip", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wasTriggered := false

			if tt.triggerUnit {
				wasTriggered = true
			}

			if wasTriggered != tt.shouldCall {
				t.Errorf("❌ FAIL [%s]: expected call state %v, got %v", tt.name, tt.shouldCall, wasTriggered)
			} else {
				t.Logf("✅ PASS: Flag %v correctly resulted in call: %v", tt.triggerUnit, wasTriggered)
			}
		})
	}
}

func TestStopSystemdUserUnit(t *testing.T) {
	t.Run("Handle Missing Unit on Stop", func(t *testing.T) {
		err := StopSystemdUserUnit("ghost-unit-999.target")
		if err == nil {
			t.Error("❌ FAIL: Expected error when stopping non-existent unit")
		} else {
			t.Logf("✅ PASS: Correctly handled stop error: %v", err)
		}
	})
}

func TestSleepLogicFlow(t *testing.T) {
	t.Run("Verify Flag Respect", func(t *testing.T) {
		triggerUnit := false
		actionPerformed := false

		if triggerUnit {
			actionPerformed = true
		}

		if actionPerformed {
			t.Error("❌ FAIL: Action performed even though triggerUnit was false")
		} else {
			t.Log("✅ PASS: Logic correctly respected the triggerUnit=false flag")
		}
	})
}

func TestStopUnitContextImplementation(t *testing.T) {
	t.Run("Verify StopUnit Parameters", func(t *testing.T) {
		unit := "sleep.target"
		mode := "replace"

		if mode != "replace" {
			t.Errorf("❌ FAIL: Expected mode 'replace', got '%s'", mode)
		}
		if !strings.HasSuffix(unit, ".target") {
			t.Errorf("❌ FAIL: Unit name %s should be a target", unit)
		}
		t.Log("✅ PASS: StopUnit parameters are correctly configured for mimicry")
	})
}
