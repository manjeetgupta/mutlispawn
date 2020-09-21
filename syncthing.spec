Name:           syncthing
Version:        1.0
Release:        1%{?dist}
Summary:        Synchronization in p2p in golang
License:        GPLv3+

%description
Syncthing application for synchronization in the background.

%prep
rm -rf %{_sourcedir}/*
install /root/Downloads/syncthing/syncthing-linux-amd64-v1.6.1/syncthing  %{_sourcedir}
install /root/Downloads/temp/syncthing@.service %{_sourcedir}
install /root/Downloads/temp/amsmsync.sh %{_sourcedir}

%install
cd %{_sourcedir}/
mkdir -p %{buildroot}/%{_prefix}/bin/
mkdir -p %{buildroot}/%{_sysconfdir}/systemd/system/
install %{name} %{buildroot}/%{_prefix}/bin/
install amsmsync.sh %{buildroot}/%{_prefix}/bin/
install syncthing@.service %{buildroot}/%{_sysconfdir}/systemd/system/

%files
%attr(0744, root, root) %{_prefix}/bin/%{name}
%attr(0744, root, root) %{_prefix}/bin/amsmsync.sh
%attr(0744, root, root) %{_sysconfdir}/systemd/system/syncthing@.service


%post 
systemctl enable syncthing@root.service
systemctl daemon-reload
systemctl start syncthing@root.service
sleep 15
systemctl stop syncthing@root.service
sleep 5
sed -i 's/<apikey>.*<\/apikey>/<apikey>manjeettest<\/apikey>/' /root/.config/syncthing/config.xml
sleep 2
sed -i 's/rescanIntervalS=.*/rescanIntervalS=\"5\" fsWatcherEnabled=\"false\" fsWatcherDelayS=\"10\" ignorePerms=\"false\" autoNormalize=\"true\">/' /root/.config/syncthing/config.xml
sleep 2
sed -i 's/SELINUX=.*/SELINUX=disabled/' /etc/selinux/config
sleep 2
systemctl stop firewalld.service
systemctl disable firewalld.service


%preun
systemctl stop syncthing@root.service
systemctl disable syncthing@root.service
systemctl daemon-reload
#rm -rf /root/.config/%{name}

%clean
rm -rf $RPM_BUILD_ROOT

%changelog
*Fri Aug 28 2020 Manjeet Gupta <manjeetgupta6@gmail.com> -1.0-1
-Initial version of the package

