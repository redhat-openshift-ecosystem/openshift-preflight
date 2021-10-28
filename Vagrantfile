Vagrant.configure("2") do |config|
  config.vm.box = "generic/fedora34"
  config.vm.synced_folder ".", "/home/vagrant/preflight"
  config.vm.provision "shell", inline: <<-SHELL
    dnf -y update
   
    dnf -y install \
    podman \
    buildah \
    jq \
    make \
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
    crun \
    containers-common \
    openscap-containers
  
    curl -L https://golang.org/dl/go1.16.8.linux-amd64.tar.gz --output go1.16.8.linux-amd64.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.16.8.linux-amd64.tar.gz
    rm go1.16.8.linux-amd64.tar.gz
    curl -L https://mirror.openshift.com/pub/openshift-v4/clients/oc/latest/linux/oc.tar.gz --output oc.tar.gz
    tar -C /usr/local/bin -xzf oc.tar.gz
    rm oc.tar.gz
    export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)
    export OS=$(uname | awk '{print tolower($0)}')
    export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.14.0
    curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
    chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk    
    echo "PATH=/usr/local/go/bin:$PATH" >> /home/vagrant/.bashrc
    echo "PATH=/usr/local/go/bin:$PATH" >> /root/.bashrc
  SHELL
end
