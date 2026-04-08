
Name:           systemd-lock-handler
Version:        %{?version}%{!?version:6}
Release:        %autorelease
Summary:        Systemd user service for lock/unlock events
License:        ISC
URL:            https://github.com/Infiniti151/systemd-lock-handler
Source0:        systemd-lock-handler.tar.gz
BuildRequires:  systemd-rpm-macros
BuildRequires:  go-rpm-macros
BuildRequires:  compiler(go-compiler)

%description
A systemd user service to handle lock/unlock events.

%prep
%setup -q -c -T
tar -xf %{SOURCE0}

%build
export PATH="/tmp/go-home/go/bin:$PATH"
go build -ldflags '-s -w' -o systemd-lock-handler main.go

%check
export PATH="/tmp/go-home/go/bin:$PATH"
go test -v ./...

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_userunitdir}
install -m 755 systemd-lock-handler %{buildroot}%{_bindir}/
install -m 644 dist/systemd-lock-handler.service.ready %{buildroot}%{_userunitdir}/systemd-lock-handler.service
install -m 644 dist/*.target %{buildroot}%{_userunitdir}/

%files
%license LICENCE
%{_bindir}/systemd-lock-handler
%{_userunitdir}/*.service
%{_userunitdir}/*.target

%post
%systemd_user_post systemd-lock-handler.service

%preun
%systemd_user_preun systemd-lock-handler.service

%postun
%systemd_user_postun_with_restart systemd-lock-handler.service

%changelog
%autochangelog
