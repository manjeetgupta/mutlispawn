Name:           amsmp2p
Version:        1.0
Release:        1%{?dist}
Summary:        Peer to Peer AMSM implementation in golang
License:        GPLv3+
Source0: 	amsmp2p-%{version}.tar.gz        
Requires:       syncthing

%description
Peer to Peer AMSM implementation in golang without using any middleware.
It utilizes Syncthing application for synchronization in the background.

%prep
rm -rf %{_sourcedir}/*
install /root/Downloads/temp/%{name}  %{_sourcedir}
install /root/Downloads/temp/%{name}.conf %{_sourcedir}
install /root/Downloads/temp/readme %{_sourcedir}
install /root/Downloads/temp/documentation.pdf %{_sourcedir}
install /root/Downloads/temp/DeploymentConfiguration.xml %{_sourcedir}
install /root/Downloads/temp/amsmp2p@.service %{_sourcedir}

%install
cd %{_sourcedir}/
%{__mkdir_p} %{buildroot}/%{getenv:HOME}/.config/AMSM/
%{__mkdir_p} %{buildroot}/%{_prefix}/local/bin/AMSM/
%{__mkdir_p} %{buildroot}/%{_prefix}/bin/
%{__mkdir_p} %{buildroot}/%{_sysconfdir}/systemd/system/
install DeploymentConfiguration.xml readme documentation.pdf %{buildroot}/%{_prefix}/local/bin/AMSM/
install %{name} %{buildroot}/%{_prefix}/bin/
install %{name}.conf %{buildroot}/%{getenv:HOME}/.config/AMSM/
install amsmp2p@.service %{buildroot}/%{_sysconfdir}/systemd/system/

%files
%attr(0744, root, root) %{_prefix}/bin/%{name}
%attr(0744, root, root) %doc %{_prefix}/local/bin/AMSM/readme
%attr(0744, root, root) %doc %{_prefix}/local/bin/AMSM/documentation.pdf
%attr(0744, root, root) %config(noreplace) %{_prefix}/local/bin/AMSM/DeploymentConfiguration.xml
%attr(0744, root, root) %config(noreplace) %{getenv:HOME}/.config/AMSM/%{name}.conf
%attr(0744, root, root) %{_sysconfdir}/systemd/system/amsmp2p@.service

%pre
mkdir -p /usr/local/bin/AMSM

%post 
systemctl enable amsmp2p@root.service
systemctl daemon-reload

%preun
systemctl stop amsmp2p@root.service
systemctl kill amsmp2p@root.service
systemctl disable amsmp2p@root.service
systemctl daemon-reload

#%postun
#rm -rf %{_prefix}/local/bin/AMSM

%clean
rm -rf $RPM_BUILD_ROOT

%changelog
*Fri Aug 28 2020 Manjeet Gupta <manjeetgupta6@gmail.com> -1.0-1
-Initial version of the package

