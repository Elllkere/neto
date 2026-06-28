# neto

`neto` is an OpenWrt/ImmortalWrt-first pre-sing-box policy router. Routing
decisions are made with firewall4/nftables before traffic reaches sing-box.
sing-box is used only as the FakeIP DNS owner, TProxy inbound backend, and proxy
outbound executor.

Russian README: [README.md](README.md)

## Install

```sh
sh -c "$(wget -O- https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh)"
```

or:

```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh)"
```

The installer downloads `neto-openwrt-embedded.tar.gz` from:

```text
https://github.com/elllkere/neto/releases/latest/download/
```

Override the archive base URL with `NETO_BASE_URL` when using a mirror.

## Uninstall

Normal uninstall keeps `/etc/config/neto` and `/etc/neto` for later reinstall:

```sh
/usr/share/neto/uninstall.sh
```

Full removal including config:

```sh
/usr/share/neto/uninstall.sh --purge
```

The script stops the service, reverts neto-owned dnsmasq settings, and removes
installed neto binaries, LuCI files, runtime files, and provider cache.

For the public GitHub install command to work, upload
`dist/neto-openwrt-embedded.tar.gz` as a GitHub Release asset named exactly:

```text
neto-openwrt-embedded.tar.gz
```

## Core Model

Bad model:

```text
LAN all traffic -> sing-box -> direct/proxy
```

neto model:

```text
LAN DNS -> dnsmasq -> netod -> selected sing-box DNS listener
LAN traffic -> nft decides before sing-box
```

Direct/bypass traffic must not enter sing-box.

## Status

Supported:

- OpenWrt 23.05+
- OpenWrt 24.10+
- OpenWrt 25.12+
- ImmortalWrt 25.12+
- firewall4/nftables
- IPv4 routing

Not in v1:

- IPv6 routing
- fw3/iptables
- transparent TCP/UDP proxy inside netod
- custom FakeIP allocator
- `.ipk` packaging

## Requirements

Minimum:

- OpenWrt/ImmortalWrt with firewall4 / nftables
- IPv4 LAN
- `sing-box` package or a compatible `sing-box` binary
- 128 MB RAM
- 25 MB free flash/overlay after dependencies
- 30 MB free `/tmp` during install/upgrade

Recommended:

- 256 MB RAM or more
- 40 MB+ free flash/overlay, especially when installing `sing-box` as a package
- ARMv7/ARM64/MIPS 24Kc-class router CPU or better

Current embedded archive is about 7 MB compressed and about 19 MB unpacked in
`/tmp` because it carries `netod` binaries for multiple CPU architectures.
Installed neto without `sing-box` takes about 5 MB of flash.

## DNS Semantics

- Domain proxy rules in `custom` mode use FakeIP.
- Provider/CIDR/IP rules use real DNS so nftables can match real destination
  IPs.
- Direct rules and direct clients always use real DNS.
- `routing_mode=global` uses real DNS by default.
- Mixed domain + provider/CIDR/IP rules are allowed; domain matchers are DNS
  phase only, provider/CIDR/IP matchers are packet/nft phase only.
- `proto`, `src_port`, and `dst_port` are packet/nft phase only and never affect
  DNS/FakeIP matching.

## Docs

The main docs are currently maintained in Russian:

- [Architecture](docs/ARCHITECTURE.md)
- [Router testing](docs/ROUTER_TESTING.md)
- [Outbounds](docs/OUTBOUNDS.md)
- [Testing](docs/TESTING.md)
- [Decisions](docs/DECISIONS.md)
- [Roadmap](docs/ROADMAP.md)
- [Release process](docs/RELEASE.md)
