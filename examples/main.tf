terraform {
  required_providers {
    upcloud = {
      source  = "terraform.local/local/upcloud"
      version = "0.1.0"
    }
  }
}

provider "upcloud" {
  username = "username" # use your username or export UPCLOUD_USERNAME
  password = "password" # use your password or export UPCLOUD_PASSWORD
}

resource "upcloud_server" "example" {
  hostname = "hostname-name"
  zone     = "de-fra1"

  network_interface {
    ip_address_family = "IPv6"
  }

  network_interface {
    ip_address_family = "IPv6"
  }
}
