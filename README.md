systemd-lock-handler
====================

[![Build](https://img.shields.io/github/actions/workflow/status/Infiniti151/systemd-lock-handler/build.yml?branch=main&style=for-the-badge&logo=github-actions&logoColor=white&label=Build&color=%23007808)](https://github.com/Infiniti151/systemd-lock-handler/actions/workflows/build.yml) [![COPR Build Status](https://img.shields.io/badge/dynamic/json?url=https://copr.fedorainfracloud.org/api_3/build/list/%3Fownername%3Dinfiniti151%26projectname%3Dsystemd-lock-handler%26packagename%3Dsystemd-lock-handler%26limit%3D1&query=$.items[0].state&label=COPR&style=for-the-badge&logo=fedora&logoColor=white&color=%2351A2DA)](https://copr.fedorainfracloud.org/coprs/infiniti151/systemd-lock-handler/package/systemd-lock-handler/) [![Latest Release](https://img.shields.io/github/v/release/Infiniti151/systemd-lock-handler?style=for-the-badge&logo=github&color=red)](https://github.com/Infiniti151/systemd-lock-handler/releases) [![Downloads](https://img.shields.io/github/downloads/Infiniti151/systemd-lock-handler/total.svg?style=for-the-badge&logo=github&color=E38A27)](https://github.com/Infiniti151/systemd-lock-handler/releases) [![Go Version](https://img.shields.io/github/go-mod/go-version/Infiniti151/systemd-lock-handler?style=for-the-badge&logo=go&logoColor=white&color=purple)](go.mod) [![GPG Signed](https://img.shields.io/badge/GPG-Signed-fedcba?style=for-the-badge&logo=gnuprivacyguard&logoColor=white&color=yellow)](https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/public.key) [![License](https://img.shields.io/badge/License-ISC-blue.svg?style=for-the-badge&logo=spdx&logoColor=white&color=pink)](LICENCE)

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

### Fedora
1. **Enable COPR:**
   ```bash
   sudo dnf copr enable infiniti151/systemd-lock-handler

2. **Install package:**
   ```bash
   sudo dnf install systemd-lock-handler

### Other Redhat based distros

1. **Import the public key:**
   ```bash
   curl -L https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/public.key | sudo rpm --import -

2. **Install RPM:**
   ```bash
   sudo dnf install $(curl -s https://api.github.com/repos/Infiniti151/systemd-lock-handler/releases/latest | grep "browser_download_url.*rpm" | cut -d '"' -f 4)

### Debian based distros

1. **Import the public key:**
   ```bash
   curl -L https://github.com/Infiniti151/systemd-lock-handler/releases/latest/download/public.key | gpg --dearmor | sudo tee /usr/share/keyrings/infiniti151-archive-keyring.gpg > /dev/null

2. **Install DEB:**
   ```bash
   curl -sL -o /tmp/lock-handler.deb $(curl -s https://api.github.com/repos/Infiniti151/systemd-lock-handler/releases/latest | grep "browser_download_url.*deb" | cut -d '"' -f 4) \
   && sudo apt install /tmp/lock-handler.deb \
   && rm /tmp/lock-handler.deb

## 🛠️ Manual Installation

You can manually build and install:

    git clone https://github.com/Infiniti151/systemd-lock-handler.git
    cd systemd-lock-handler
    make install

Usage
-----

The service itself must be enabled for the current user (for package installs):

    systemctl --user enable --now systemd-lock-handler.service

Additionally, service files must be created and enabled for any service that should start when state changes occur.

For example, `enabling` this service file would run `swaylock` when `logind`
locks the session:

    [Unit]
    Description=Screen locker for Wayland
    # If swaylock exits cleanly, unlock the session:
    OnSuccess=unlock.target
    # When lock.target is stopped, stop this too:
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

### Target Binding Best Practices

* `PartOf=lock.target`: Use this for services that must stop the moment you unlock your screen (or transition to sleep/wake), such as a "Do Not Disturb" mode or a script that pauses background syncs.
* `PartOf=sleep.target`: Use this for pre-suspend tasks that should cleanly terminate once the state moves away from sleep.
* `WantedBy=lock.target`: Services that trigger whenever the session locks (e.g., screen lockers, dimming keyboard LEDs, pausing media).
* `WantedBy=unlock.target`: Services that run upon unlocking the screen (e.g., re-enabling dGPU power, refreshing notifications).
* `WantedBy=sleep.target`: Pre-suspend services (e.g., saving session state or pausing sync routines before the kernel freezes).
* `WantedBy=wake.target`: Post-resume tasks that execute as soon as the system resumes from sleep (e.g., reconnecting network drives, resetting Bluetooth adapters, restoring display states).

## Target Triggers

### Locking
Lock your session using `loginctl lock-session`.
Starts `lock.target` and stops other targets.

### Unlocking
Unlock your session using `loginctl unlock-session`.
Starts `unlock.target` and stops `lock.target` (stopping any units with `PartOf=lock.target`).

### Suspending
Sleep your device using `systemctl suspend`.
Starts `sleep.target` before the kernel suspends the system and stops other active targets.

### Resuming
When the system resumes from suspend:
Starts `wake.target` and stops `sleep.target` (stopping any units with `PartOf=sleep.target`). Transient lock/unlock events emitted by display managers during resume are automatically filtered out via an internal cooldown buffer.

## Detection Toggling

By default, detection for all events (`sleep`, `wake`, `lock`, and `unlock`) is enabled. D-Bus signal monitoring can be selectively disabled via CLI flags in a configuration file.

| Flag | Function | Default |
| :--- | :----: | :----: |
| `-sleep` | User-level suspend detection (`sleep.target`) | `true` |
| `-wake` | User-level resume detection (`wake.target`) | `true` |
| `-lock` | Lock detection (`lock.target`) | `true` |
| `-unlock` | Unlock detection (`unlock.target`) | `true` |

**Example configuration (`~/.config/systemd-lock-handler.conf`):**

To disable sleep/wake triggers and rely only on session locking:
```ini
FLAGS="-sleep=false -wake=false"
```

Reload the daemon and restart the service:
```
systemctl --user daemon-reload
systemctl --user restart systemd-lock-handler
```

The detection status for all four events is shown in the service status (```systemctl --user status systemd-lock-handler```):
![alt text](image.png)
