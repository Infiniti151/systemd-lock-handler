systemd-lock-handler
====================

[![Build](https://img.shields.io/github/actions/workflow/status/Infiniti151/systemd-lock-handler/build.yml?branch=main&style=for-the-badge&labelColor=489FC3)](https://github.com/Infiniti151/systemd-lock-handler/actions/workflows/build.yml) [![Latest Release](https://img.shields.io/github/v/release/Infiniti151/systemd-lock-handler?branch=main&style=for-the-badge&labelColor=489FC3&color=red)](https://github.com/Infiniti151/systemd-lock-handler/releases) [![Downloads](https://img.shields.io/github/downloads/Infiniti151/systemd-lock-handler/total.svg?Label=Downloads&style=for-the-badge&labelColor=489FC3&color=E38A27)](https://github.com/Infiniti151/systemd-lock-handler/releases) [![Go Version](https://img.shields.io/github/go-mod/go-version/Infiniti151/systemd-lock-handler?branch=main&style=for-the-badge&labelColor=489FC3&color=purple)](go.mod) [![GPG Signed](https://img.shields.io/badge/GPG-Signed-ffd700?branch=main&style=for-the-badge&labelColor=489FC3)](https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/public.key) [![License](https://img.shields.io/github/license/Infiniti151/systemd-lock-handler?style=for-the-badge&labelColor=489FC3&color=gray)](LICENSE)

`logind` (part of systemd) emits events when the system is locked, unlocked or
goes into sleep.

These events however, are simple D-Bus events, and don't actually run anything.
There are no facilities for users to easily _run_ anything on these events
either (e.g.: a screen locker).

`systemd-lock-handler` is a small, lightweight helper fills this gap.

When the system is either locked, unlocked, or about to go into sleep, this
service will start the `lock.target`, `unlock.target` and `sleep.target`
systemd user targets respectively.

When the system is unlocked, `lock-target` will be stopped.

Any service can be configured to start with any of these targets:

- A screen locker.
- A service that keeps the screen off after 15 seconds of inactivity.
- A service that turns the volume to 0%.
- ...

Note that systemd already has a `sleep.target`, however, that's a system-level
target, and your user-level units can't rely on it. The one included in this
package does not conflict, but rather compliments that one.

Installation
------------

## 🛡️ Verified Package Installation

This package is digitally signed. To verify the integrity of the download:
### Redhat based systems

1. **Import the public key:**
   ```bash
   curl -L https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/public.key | sudo rpm --import -

2. **Install RPM:**
   ```bash
   sudo dnf install https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/systemd-lock-handler-v<version>.rpm

### Debian based systems

1. **Import the public key:**
   ```bash
   curl -L https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/public.key | gpg --dearmor | sudo tee /usr/share/keyrings/infiniti151-archive-keyring.gpg > /dev/null

3. **Install DEB:**
   ```bash
   curl -L -O https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/systemd-lock-handler-v<version>.deb
   sudo apt install ./systemd-lock-handler-v<version>.deb

## 🛠️ Manual Installation

You can manually build and install:

    git clone https://github.com/Infiniti151/systemd-lock-handler.git
    cd systemd-lock-handler
    make install

Usage
-----

The service itself must be enabled for the current user (for package installs):

    systemctl --user enable --now systemd-lock-handler.service

Additionally, service files must be created and enabled for any service that
should start when the system is locked.

For example, `enabling` this service file would run `swaylock` when `logind`
locks the session and before the system goes to sleep:

    [Unit]
    Description=Screen locker for Wayland
    # If swaylock exits cleanly, unlock the session:
    OnSuccess=unlock.target
    # When lock.target is stopped, stops this too:
    PartOf=lock.target
    # Delay lock.target until this service is ready:
    After=lock.target

    [Service]
    # systemd will consider this service started when swaylock forks...
    Type=forking
    # ... and swaylock will fork only after it has locked the screen.
    ExecStart=/usr/bin/swaylock -f
    # If swaylock crashes, always restart it immediately:
    Restart=on-failure
    RestartSec=0

    [Install]
    WantedBy=lock.target

Specifying `PartOf=lock.target` indicates to systemd that this service should
be stopped if `lock.target` is stopped. This is even more important for
services that _aren't_ the screen locker, since this setting means they'll get
stopped when the system is unlocked.

Specifying `WantedBy=lock.target` will have this service run when locking
**or** sleeping the system.

Specifying `WantedBy=sleep.target` will have this service run only when
sleeping the system. Note that the service will continue running after
waking up from sleep.

## Locking

Lock your session using `loginctl lock-session`.

This will mark the session as locked, and start `lock.target` along with any
services that are `WantedBy` it.

## Unlocking

Unlock your session using `loginctl unlock-session`.

This will mark the session as unlocked, start `unlock.target`, and stop
`lock.target`. 

Service that are marked `PartOf=lock.target` will be stopped when `lock.target`
stops.

## Suspending

Sleep your device using `systemctl suspend`.

This will start `sleep.target` along with any services that are `WantedBy` it.
This will happen _before_ the system is suspended.

## Detection Toggling

Be default, detection for all events (sleep, lock, and unlock) is enabled. This detection can be toggled individually via flags (-sleep, -lock, -unlock). These flags need to be added to the service file at /usr/lib/systemd/user/systemd-lock-handler.service. Example of lock and unlock detection turned off:

```
[Service]
Slice=session.slice
ExecStart=/usr/bin/systemd-lock-handler -lock=false -unlock=false
```

After adding the flags, reload the daemon and restart the service:
```
systemctl --user daemon-reload
systemctl --user restart systemd-lock-handler
```

The detection status for all three events is shown in service status (```systemctl --user status systemd-lock-handler```):
![alt text](image.png)




