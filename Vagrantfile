# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/focal64"
  
  config.vm.hostname = "fantasy-frc"
  
  config.vm.network "private_network", ip: "192.168.56.10"
  
  config.vm.provider "virtualbox" do |vb|
    vb.memory = "2048"
    vb.cpus = 2
    vb.name = "fantasy-frc-dev"
  end
  
  config.vm.provision "shell", inline: <<-SHELL
    echo "Updating package list..."
    apt-get update -y
    
    echo "Installing Python and Ansible..."
    apt-get install -y python3 python3-pip
    pip3 install ansible
    
    echo "Installing PostgreSQL..."
    apt-get install -y postgresql postgresql-contrib
    systemctl enable postgresql
    systemctl start postgresql
    
    echo "Installing Redis..."
    apt-get install -y redis-server
    systemctl enable redis-server
    systemctl start redis-server
    
    echo "Installing Go..."
    curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz -o /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    export PATH=$PATH:/usr/local/go/bin
    
    echo "Creating fantasyfrc user..."
    useradd -r -s /bin/false -d /opt/fantasy-frc -M fantasyfrc || true
    mkdir -p /opt/fantasy-frc
    chown fantasyfrc:fantasyfrc /opt/fantasy-frc
    
    echo "Setup complete!"
  SHELL
  
  config.vm.provision "ansible" do |ansible|
    ansible.playbook = "/vagrant/deploy/ansible/playbook.yml"
    ansible.inventory_path = "/vagrant/deploy/ansible/inventory.ini"
    ansible.limit = "all"
    ansible.extra_vars = {
      vault_db_password: "testpassword",
      vault_tba_token: "test_tba_token",
      vault_tba_webhook_secret: "test_webhook_secret",
      vault_session_secret: "test_session_secret_$(date +%s)"
    }
  end
end
