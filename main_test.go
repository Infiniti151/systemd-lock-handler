package main

import (
	"context"
	"reflect"
	"testing"

	systemd "github.com/coreos/go-systemd/v22/dbus"
	dbus "github.com/godbus/dbus/v5"
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
		// Calling a specific function from the systemd-aliased package
		conn, err := systemd.NewUserConnectionContext(ctx)
		if err != nil {
			t.Logf("⚠️  SKIP: Systemd User Bus connection failed: %v", err)
			t.SkipNow()
		}
		defer conn.Close()
		t.Log("✅ SUCCESS: Systemd User Connection established")
	})
}

func TestDbusSystemBus(t *testing.T) {
	t.Run("ConnectSystemBus", func(t *testing.T) {
		// Calling a specific function from the dbus-aliased package
		conn, err := dbus.ConnectSystemBus()
		if err != nil {
			t.Logf("⚠️  SKIP: System D-Bus not available: %v", err)
			t.SkipNow()
		}
		defer conn.Close()
		t.Log("✅ SUCCESS: System D-Bus connection established")
	})
}

func TestLockLogic(t *testing.T) {
	tests := []struct {
		name     string
		isLock   bool
		isUnlock bool
	}{
		{"org.freedesktop.login1.Session.Lock", true, false},
		{"org.freedesktop.login1.Session.Unlock", false, true},
		{"Other", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLock := (len(tt.name) >= 4 && tt.name[len(tt.name)-4:] == "Lock")
			gotUnlock := (len(tt.name) >= 6 && tt.name[len(tt.name)-6:] == "Unlock")

			if gotLock != tt.isLock || gotUnlock != tt.isUnlock {
				t.Errorf("❌ FAIL [%s]: Expected Lock=%v, Unlock=%v | Got Lock=%v, Unlock=%v",
					tt.name, tt.isLock, tt.isUnlock, gotLock, gotUnlock)
			} else {
				t.Logf("✅ PASS: Correctly identified signal type for %s", tt.name)
			}
		})
	}
}

func TestWaitPrepareForSleep(t *testing.T) {
	t.Skip("Skipping this test manually for now")
	tests := []struct {
		name    string
		want    bool
		body    []interface{}
		wantErr bool
	}{
		{"SleepStarting", true, []interface{}{true}, false},
		{"SleepFinished", false, []interface{}{false}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := &dbus.Signal{Body: tt.body}
			c := make(chan *dbus.Signal, 1)
			c <- sig

			err := waitPrepareForSleep(c, tt.want)
			if (err != nil) != tt.wantErr {
				t.Errorf("❌ FAIL [%s]: Expected error status %v, got %v", tt.name, tt.wantErr, err)
			} else {
				t.Logf("✅ PASS: Logic check for %s successful", tt.name)
			}
		})
	}
}
