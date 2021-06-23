Vagrant.configure("2") do |config|
  config.vm.box = "generic/fedora33"
  config.vm.synced_folder ".", "/home/vagrant/preflight"
  config.vm.provision "shell", inline: <<-SHELL
    dnf -y update
   
    dnf -y install \
    podman \
    buildah \
    skopeo \
    jq \
    make \
    golang \
    bats \
    btrfs-progs-devel \
    device-mapper-devel \
    glib2-devel \
    gpgme-devel \
    libassuan-devel \
    libseccomp-devel \
    git \
    bzip2 \
    go-md2man \
    runc \
    containers-common
    curl -L https://golang.org/dl/go1.16.3.linux-amd64.tar.gz --output go1.16.3.linux-amd64.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.16.3.linux-amd64.tar.gz
    echo "PATH=$PATH:/usr/local/go/bin" >> /home/vagrant/.bashrc
    echo "PATH=$PATH:/usr/local/go/bin" >> /root/.bashrc
  SHELL
end
