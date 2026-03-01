systemd-lock-handler
====================

[![Build](https://img.shields.io/github/actions/workflow/status/Infiniti151/systemd-lock-handler/build.yml?branch=main&event=schedule&style=for-the-badge&labelColor=489FC3)](https://github.com/Infiniti151/systemd-lock-handler/actions/workflows/build.yml) [![Downloads](https://img.shields.io/github/downloads/Infiniti151/systemd-lock-handler/total.svg?Label=Downloads&style=for-the-badge&labelColor=489FC3&color=E38A27)](https://github.com/Infiniti151/systemd-lock-handler/releases)

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

## Package install

Install the latest release from the releases page (RPM/DEB)

## Manual install

You can manually build and install:

    git clone https://github.com/Infiniti151/systemd-lock-handler.git
    cd systemd-lock-handler
    make build
    sudo make install

Usage
-----

The service itself must be enabled for the current user:

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

