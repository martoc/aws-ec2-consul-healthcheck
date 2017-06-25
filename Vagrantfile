# -*- mode: ruby -*-
# vi: set ft=ruby :

$script = <<SCRIPT
yum groupinstall "Development Tools"
curl -O https://storage.googleapis.com/golang/go1.8.3.linux-amd64.tar.gz
tar -xvf go1.8.3.linux-amd64.tar.gz
SCRIPT

Vagrant.configure(2) do |config|
  config.vm.box = "centos/7"
  config.vm.provision "shell", inline: $script
end
