%define booking_repo extras

%global debug_package   %{nil}
%global import_path     github.com/tdevelioglu/prometheus-nginx-log-exporter
%global gopath          %{_datadir}/gocode

%global _extdir ext/redhat

Name:           prometheus-nginx-log-exporter
Version:        0.1.0
Release:        1%{?dist}
Summary:        Prometheus exporter for Nginx logs
License:        MIT
URL:            http://%{import_path}
Source0:        https://%{import_path}/archive/%{version}/%{name}-%{version}.tar.gz
BuildRequires:  git
BuildRequires:  golang
BuildRequires:  systemd
Requires:       systemd

%description
%{summary}

%prep
%setup

%build
mkdir _build
pushd _build

mkdir -p src/$(dirname %{import_path})
ln -s $(dirs +1 -l) src/%{import_path}
export GOPATH=$(pwd):%{gopath}

go get %{import_path}
go build -v -a %{import_path}

popd

%install
install -Dp -m 0755 _build/%{name} %{buildroot}%{_sbindir}/%{name}
install -Dp -m 0644 %{_extdir}/%{name}.service %{buildroot}%{_unitdir}/%{name}.service
install -Dp -m 0644 %{_extdir}/%{name}.sysconfig %{buildroot}%{_sysconfdir}/sysconfig/%{name}
install -Dp -m 0644 %{_extdir}/%{name}.yaml %{buildroot}%{_sysconfdir}/%{name}/config.yaml
install -d  -m 0755 %{buildroot}%{_sysconfdir}/%{name}/apps.d
install -d  -m 0711 %{buildroot}%{_var}/empty/%{name}

%clean
rm -rf $RPM_BUILD_ROOT

%pre
getent group %{name} >/dev/null || groupadd -r %{name} || :
getent passwd %{name} >/dev/null || \
    useradd -c "Prometheus Nginx log exporter" -g %{name} \
    -s /sbin/nologin -r -d / %{name} 2> /dev/null || :

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%defattr(-,root,root,-)
%dir %attr(0711,root,root) %{_var}/empty/sshd
%dir %{_sysconfdir}/%{name}/apps.d
%{_sbindir}/%{name}
%{_unitdir}/%{name}.service
%config(noreplace) %{_sysconfdir}/sysconfig/%{name}
%config(noreplace) %attr(0644, root, root) %{_sysconfdir}/%{name}/config.yaml
