# Fairwave Terraform - optional cloud breakout hub

A single AWS `t3.medium` instance acting as the mesh rendezvous/breakout hub
for a Fairwave deployment: it holds a WireGuard hub config, accepts `51820/udp`
from mesh peers, and exposes `443/tcp` for the HTTP rendezvous service.

This is a **stub** - a minimal, real starting point. It does not create a VPC,
NAT, or IAM hardening; adapt the provider configuration to your own
account/tooling before use.

## Usage

```console
cp terraform.tfvars.example terraform.tfvars   # or set TF_VAR_* vars
terraform init
terraform plan
terraform apply
```

Required variables: `wg_private_key` (base64 WireGuard private key, generated
by the control plane - never commit it), `admin_ssh_key` (public key for the
instance).

## Resources

- `aws_instance.hub` - t3.medium, Debian 12 (bookworm) AMI
- `aws_security_group.hub` - `51820/udp` + `443/tcp` ingress, egress all
- `user_data` - installs WireGuard, writes `/etc/wireguard/wg0.conf` from
  `wg_private_key`, enables IP forwarding, starts the interface

## Notes

- The hub is a rendezvous point only; it carries no SIM material and does not
  store keys beyond its own WireGuard private key.
- See `docs/peering/rendezvous.md` for the full mesh design.
