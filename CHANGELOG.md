# Changelog

## [19] - 2026-08-20
- Upgraded Go compiler environment from **1.26.6** to **1.27.0**

## [18] - 2026-08-14
- Upgraded Go compiler environment from **1.26.5** to **1.26.6**

## [17] - 2026-07-30
- Added `wake.target` for handling post-resume tasks
- Refactored target state transitions to use exclusive target switching
- Updated release workflow to extract notes from `CHANGELOG.md` and trigger COPR builds

## [16] - 2026-07-09
- Updated `x/sys` from v0.46.0 to v0.47.0

## [15] - 2026-07-08
- Upgraded Go compiler environment from 1.26.4 to 1.26.5

## [14] - 2026-06-09
- Updated `x/sys` from 0.45.0 to 0.46.0

## [13] - 2026-06-03
- Upgraded Go compiler environment from 1.26.3 to 1.26.4

## [12] - 2026-05-22
- Updated `x/sys` from 0.44.0 to 0.45.0

## [11] - 2026-05-19
- Updated to use a config file instead of a systemd drop-in file for setting flags

## [10] - 2026-05-09
- Updated `x/sys` from 0.43.0 to 0.44.0

## [9] - 2026-05-08
- Upgraded Go compiler environment from 1.26.2 to 1.26.3

## [8] - 2026-04-09
- Updated `x/sys` from 0.42.0 to 0.43.0

## [7] - 2026-04-08
- Upgraded Go compiler environment from 1.26.1 to 1.26.2

## [6] - 2026-04-06
- Added block-sleep-lock flag to filter out lock/unlock events from suspend/resume.
- Decoupled sleep.target from lock.target

## [5] - 2026-03-11
- Replaced D-Bus Lock signals with LockedHint property for more reliable detection
- Added command-line flags to toggle detection for events
- Added logic to debounce duplicate signals (especially duplicate lock signals during sleep event)

## [4] - 2026-03-09
- Updated `x/sys` from 0.41.0 to 0.42.0

## [3] - 2026-03-07
- Upgraded Go compiler environment from 1.26.0 to 1.26.1

## [2] - 2026-03-03
- Updated `x/sys` from 0.27.0 to 0.41.0

## [1] - 2026-03-01
- Initial release
