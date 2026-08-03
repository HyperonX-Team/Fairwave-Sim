# Fairwave cloud breakout hub - AWS stub. See README.md.
terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

data "aws_ami" "debian_bookworm" {
  most_recent = true
  owners      = ["136693071363"] # Debian project

  filter {
    name   = "name"
    values = ["debian-12-*"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

resource "aws_security_group" "hub" {
  name        = "${var.project}-hub"
  description = "Fairwave mesh hub: wg 51820/udp + rendezvous 443/tcp"
  vpc_id      = var.vpc_id

  ingress {
    description = "WireGuard mesh"
    from_port   = 51820
    to_port     = 51820
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "Rendezvous / HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "hub" {
  ami           = data.aws_ami.debian_bookworm.id
  instance_type = var.instance_type
  vpc_security_group_ids = [aws_security_group.hub.id]
  key_name      = var.ssh_key_name

  user_data = <<-EOT
    #!/usr/bin/env bash
    set -euo pipefail
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y wireguard
    sysctl -w net.ipv4.ip_forward=1
    echo 'net.ipv4.ip_forward=1' > /etc/sysctl.d/99-fairwave.conf
    umask 077
    cat > /etc/wireguard/wg0.conf <<EOF
    [Interface]
    Address = ${var.hub_wg_address}
    ListenPort = ${var.wg_port}
    PrivateKey = ${var.wg_private_key}

    # Peer entries are appended by fairwave-control when nodes enroll.
    EOF
    systemctl enable wg-quick@wg0
    systemctl start wg-quick@wg0
  EOT

  tags = {
    Name    = "${var.project}-hub"
    Project = var.project
  }
}

resource "aws_eip" "hub" {
  count    = var.allocate_eip ? 1 : 0
  instance = aws_instance.hub.id
  domain   = "vpc"

  tags = {
    Name = "${var.project}-hub-eip"
  }
}
