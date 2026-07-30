%global debug_package %{nil}
%global _find_debuginfo_dwz_opts %{nil}
%global _build_id_links none
%global target_go_ver 1.26.5

Name:           systemd-lock-handler
Version:        16
Release:        %autorelease
Summary:        Systemd user service for lock/unlock events
License:        ISC
URL:            https://github.com/Infiniti151/%{name}
BugURL:         https://github.com/Infiniti151/%{name}/issues

Source0:        %{url}/archive/v%{version}.tar.gz

BuildRequires:  systemd-rpm-macros
BuildRequires:  curl

%description
A systemd user service to handle lock/unlock events.

%prep
%autosetup
echo "=== Extracted Source Tarball ==="

%ifarch x86_64
echo "=== Fetching Pre-built x64 Binary from GitHub Releases ==="
curl -sL "%{url}/releases/download/v%{version}/%{name}" -o %{name}

if ! file %{name} | grep -q "ELF.*x86-64"; then
    echo "ERROR: Downloaded file is not a valid x86_64 ELF binary!"
    cat %{name}
    exit 1
fi
chmod +x %{name}

%else
echo "=== Fetching Go version %{target_go_ver} for ARM64 ==="
mkdir -p ./go-home
curl -sL "https://go.dev/dl/%{target_go_ver}.linux-arm64.tar.gz" -o go-arm64.tar.gz
tar -C ./go-home -xf go-arm64.tar.gz
echo "=== Extracted Go Tarball ==="
%endif

%build
echo "=*=*=*> Running Build Phase <*=*=*="

%ifarch x86_64
echo "=== Skipping Compilation for x86_64 (Using GitHub Release Asset) ==="

%else
echo "=== Compiling from Source for %{_arch} ==="
GO_BIN=$(pwd)/go-home/go/bin/go

if [ ! -x "$GO_BIN" ]; then
    echo "ERROR: Custom Go compiler not found at $GO_BIN"
    exit 1
fi

echo "=== Go version: $($GO_BIN version) ==="

export GOTOOLCHAIN=local
export CGO_ENABLED=0

echo "=== Go environment ==="
$GO_BIN env

echo "=== Building %{name} ==="

$GO_BIN build -ldflags '-s -w' -o %{name} main.go
%endif

sed "s|{{BIN_PATH}}|%{_bindir}/%{name}|g" dist/%{name}.service > dist/%{name}.service.ready

%check
echo "=*=*=*> Running Check Phase <*=*=*="
%ifarch x86_64
echo "=== Verifying downloaded x86_64 binary execution ==="
./%{name} --help || true
%else
GO_BIN=$(pwd)/go-home/go/bin/go
export GOTOOLCHAIN=local

echo "=== Package List ==="
$GO_BIN list .

echo "=== Running Tests ==="
$GO_BIN test -v .
%endif

%install
echo "=*=*=*> Running Install Phase <*=*=*="
echo "Working directory: $(pwd)"
echo "Buildroot: %{buildroot}"

if [ ! -f %{name} ]; then
    echo "ERROR: Binary '%{name}' not found in $(pwd)!"
    ls -lh
    exit 1
fi

mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_userunitdir}

echo "=== Installing binary to %{buildroot}%{_bindir}/ ==="
install -m 755 %{name} %{buildroot}%{_bindir}/

echo "=== Installing systemd units to %{buildroot}%{_userunitdir}/ ==="
install -m 644 dist/%{name}.service.ready %{buildroot}%{_userunitdir}/%{name}.service
install -m 644 dist/*.target %{buildroot}%{_userunitdir}/

echo "=== Final Buildroot Contents ==="
find %{buildroot} -maxdepth 4 -not -path '*/.*'

echo "=*=*=*> Finished Install Phase - Moving to File Manifest <*=*=*="

%files
%license LICENCE
%doc README.md
%{_bindir}/%{name}
%{_userunitdir}/*.service
%{_userunitdir}/*.target

%post
%systemd_user_post %{name}.service

%preun
%systemd_user_preun %{name}.service

%postun
%systemd_user_postun %{name}.service

%changelog
* Thu Jul 09 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 16-1
- Updated `x/sys` from v0.46.0 to v0.47.0

* Wed Jul 08 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 15-1
- Upgraded Go compiler environment from 1.26.4 to 1.26.5

* Tue Jun 09 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 14-1
- Updated `x/sys` from 0.45.0 to 0.46.0

* Wed Jun 03 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 13-1
- Upgraded Go compiler environment from 1.26.3 to 1.26.4

* Fri May 22 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 12-1
- Updated `x/sys` from 0.44.0 to 0.45.0

* Tue May 19 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 11-1
- Updated to use a config file instead of a systemd drop-in file for setting flags

* Sat May 09 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 10-1
- Updated `x/sys` from 0.43.0 to 0.44.0

* Fri May 08 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 9-1
- Upgraded Go compiler environment from 1.26.2 to 1.26.3

* Thu Apr 09 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 8-1
- Updated `x/sys` from 0.42.0 to 0.43.0

* Wed Apr 08 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 7-1
- Upgraded Go compiler environment from 1.26.1 to 1.26.2

* Mon Apr 06 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 6-1
- Added block-sleep-lock flag to filter out lock/unlock events from suspend/resume.
- Decoupled sleep.target from lock.target

* Wed Mar 11 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 5-1
- Replaced D-Bus Lock signals with LockedHint property for more reliable detection
- Added command-line flags to toggle detection for events
- Added logic to debounce duplicate signals (especially duplicate lock signals during sleep event)

* Mon Mar 09 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 4-1
- Updated `x/sys` from 0.41.0 to 0.42.0

* Sat Mar 07 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 3-1
- Upgraded Go compiler environment from 1.26.0 to 1.26.1

* Tue Mar 03 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 2-1
- Updated `x/sys` from 0.27.0 to 0.41.0

* Sun Mar 01 2026 Infiniti151 <43163551+Infiniti151@users.noreply.github.com> - 1-1
- Initial release

