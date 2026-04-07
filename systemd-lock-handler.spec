
Name:           systemd-lock-handler
Version:        %{?_version}%{!?_version:6}
Release:        1%{?dist}
Summary:        Systemd user service for lock/unlock events
License:        ISC
URL:            https://github.com/Infiniti151/systemd-lock-handler
Source0:        systemd-lock-handler.tar.gz
BuildRequires:  golang
BuildRequires:  systemd-rpm-macros

%description
A systemd user service to handle lock/unlock events.

%prep
%setup -q -n .

%build
go build -ldflags '-s -w' -o systemd-lock-handler main.go

%check
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
