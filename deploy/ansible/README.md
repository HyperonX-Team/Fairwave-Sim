# Fairwave Ansible deployment (Debian 12 host)

Installs the Fairwave node stack on a Debian 12 host:

1. Docker Engine (`docker.io` + `docker-compose-v2` from Debian repos)
2. `fairwave` system user with docker group access
3. UFW firewall - SSH allowed, `8080/tcp` allowed from RFC1918 only,
   `51820/udp` open for the WireGuard mesh (peering)
4. `fairwave-control` and `fairwave-agent` as systemd services
5. Config files rendered from Jinja templates; **secrets only via
   `/etc/fairwave/env`** (0600, created from inventory vars, never committed)

## Usage

```console
ansible-playbook -i inventory.example.yml playbook.yml
```

## Layout

```
deploy/ansible/
├── inventory.example.yml
├── playbook.yml
└── roles/fairwave/
    ├── handlers/main.yml
    ├── tasks/main.yml
    └── templates/
        ├── fairwave-control.yaml.j2
        └── fairwave-agent.yaml.j2
```

## Secrets policy

`/etc/fairwave/env` is the only file holding credentials (agent token,
WireGuard keys). The playbook writes it from `fairwave_env` inventory vars
with mode 0600. Add the file to `inventory.example.yml` **never** - supply
it via `-e @secrets.yml` from an encrypted/ansible-vault file or an
unmanaged path.

## Firewall defaults

- `22/tcp` from anywhere (SSH - restrict via `fairwave_ufw_ssh_from`)
- `8080/tcp` from RFC1918 (internal control-plane API)
- `51820/udp` from anywhere (mesh peers reach you via UDP)

Adjust `fairwave_ufw_rules` in the playbook to taste. TX/RF is gated by the
control plane regardless of the firewall (see `deploy/scripts/rf-gate.sh`).
