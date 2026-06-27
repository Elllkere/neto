# Architecture

## Goal

`neto` is a pre-sing-box policy router for OpenWrt/ImmortalWrt. It makes routing
decisions in nftables before traffic reaches sing-box. sing-box is a backend,
not the policy authority.

The default failure mode should be direct internet routing whenever possible. If
neto is disabled or fails to apply, normal direct routing should remain intact.

## Supported Platforms

Supported:

- OpenWrt 23.05
- OpenWrt 24.10
- OpenWrt 25.12+
- ImmortalWrt 25.12+

Not supported:

- fw3
- iptables
- OpenWrt releases older than 23.05

Only firewall4/nftables is supported.

## Embedded-First Distribution

The distribution model is an embedded archive, not an `.ipk` package.

Installers should:

- detect OpenWrt/ImmortalWrt
- detect version
- require firewall4/nftables
- choose `apk` or `opkg`
- install dependencies
- detect CPU architecture/ABI
- unpack `neto-openwrt-embedded.tar.gz`
- install the correct `netod`
- prefer compatible system sing-box
- install managed sing-box to `/usr/libexec/neto/sing-box` when needed
- never overwrite `/usr/bin/sing-box`

The archive should have a top-level `neto/` directory.

## Main Components

- `cmd/netod/main.go`: CLI entrypoint.
- `internal/config`: UCI parser and config validation.
- `internal/ruleengine`: ordered rule/domain matcher logic.
- `internal/dnsproxy`: UDP/TCP DNS listener and forwarding policy.
- `internal/provider`: provider CIDR file loading.
- `internal/policy`: IPv4 CIDR normalization/dedup/collapse.
- `internal/nft`: nftables generation.
- `internal/singbox`: sing-box config generation/check support.
- `internal/tproxy`: policy routing command planning.
- `internal/status`: status/listener/routing checks.
- `embedded/files/etc/init.d/neto`: procd service.
- `embedded/files/www/luci-static/resources/view/neto`: LuCI app.

## Runtime Paths

- `/etc/config/neto`
- `/etc/init.d/neto`
- `/usr/bin/netod`
- `/usr/libexec/neto/sing-box`
- `/usr/share/neto/`
- `/tmp/neto/neto.nft`
- `/tmp/neto/sing-box.json`
- `/var/lib/neto/`

## CLI Surface

Current milestone commands:

- `netod check`
- `netod compile`
- `netod apply`
- `netod status`
- `netod debug`
- `netod run`

Planned/debug-oriented commands requested for router testing:

- `netod rules list`
- `netod test-domain 192.168.8.100 youtube.com A`

If those planned commands are not implemented yet, treat that as a roadmap gap,
not as a reason to change routing semantics.

## Lifecycle Model

The current OpenWrt init model is:

- init script runs `netod check`
- init script runs `netod compile`
- init script runs `netod apply`
- procd starts `netod run`
- procd starts selected sing-box binary with `/tmp/neto/sing-box.json`

`netod run` owns the DNS listener. sing-box owns FakeIP DNS and the TProxy
inbound.

Stop/disable should remove neto-owned nft table and TProxy policy routing state
without killing unrelated system sing-box processes.

## nft/TProxy

Generated nftables table:

- `table inet neto`
- LAN source guard first
- `direct_clients4` return before proxy rules
- `reserved4` return before proxy rules
- `proxy_clients4` source rule
- FakeIP range rule in custom mode
- ordered provider/rule sets
- default return

TProxy policy routing:

```sh
ip -4 rule add fwmark <mark> table <table>
ip -4 route add local default dev lo table <table>
```

Repeated start/reload must be idempotent.

## LAN Guard

neto must only route LAN client traffic. Generated nft must return WAN,
inbound, router self, and non-LAN prerouting traffic before any proxy/tproxy
decision.

LAN scope is represented by:

- `lan_subnets4` nft set
- optional `lan_iface` matching

## DNS/FakeIP

`netod` listens on `dns_listen`, currently `127.0.0.1:5353`, UDP and TCP.

DNS behavior:

- direct clients always use real upstream DNS
- no FakeIP for direct clients
- custom mode evaluates ordered domain rules
- global mode defaults to real DNS unless client policy forces proxy
- FakeIP A answers come from sing-box DNS
- FakeIP-matched AAAA should return prompt NODATA/empty NOERROR when IPv6
  routing is absent
- block rules may return NXDOMAIN in DNS phase

sing-box owns FakeIP allocation and FakeIP-to-domain mapping.

## Providers and Rules

Provider is data source only. Rule is routing policy only.

Provider files:

- contain IPv4 CIDRs
- are loaded by rule references
- must compile into nft interval sets
- must not generate thousands of nft rules

Rules:

- have `enabled`
- have `priority`
- have `action`: `proxy`, `direct`, `block`
- have `dns_mode`: `fakeip`, `real_ip`, `auto`
- reference provider files with `list file`
- are evaluated by ascending priority
- use first-match-wins semantics

Creating a provider must not create a rule. Creating a rule must not create a
provider.

## Outbound Profiles

Rules store an outbound tag. Built-in tags are:

- `direct`
- `blocked`

These built-ins are always generated for sing-box and must not be configured as
`config outbound` sections.

Supported native sing-box outbound profile types:

- `vless`
- `hysteria2`
- `shadowsocks`
- `trojan`

Custom outbound sections must use their own stable tags, for example
`my_vless`. The optional `label` is only a human-readable name and must not
change routing references. Outbound sections do not have an enable/disable
switch in v1; delete the section to remove a profile. `proxy_default` is
deprecated and ignored as an outbound section.

Outbound profile fields are native sing-box fields plus a small set of
homeproxy-compatible aliases accepted by the parser for migration. LuCI should
keep the table to `label`, `type`, `address`, and `port`, with advanced TLS,
REALITY, uTLS, ECH, flow, method, and transport options in the edit dialog.

Outbound profiles must not change nft routing policy. nftables still decides
which LAN client packets reach sing-box before sing-box executes the selected
outbound.

## Client Policy Model

- absent/default: follow `routing_mode`
- `proxy`: force non-reserved client TCP/UDP traffic through neto
- `direct`: hard bypass, real DNS only, no FakeIP, nft return before proxy
  rules

Old aliases may be accepted in backend for compatibility, but LuCI should write
only current policy names.

## Routing Modes

`routing_mode=custom`:

- selective routing by ordered rules
- unmatched traffic follows `default_outbound`
- in v1, `default_outbound` is only `direct`

`routing_mode=global`:

- all LAN client TCP/UDP is proxied
- direct clients bypass
- reserved/private/local destinations return/direct

## Matcher Semantics

Domain matchers are literal string operations:

- `domain_equals`: `==`
- `domain_contains`: `strings.Contains`
- `domain_starts_with`: `strings.HasPrefix`
- `domain_ends_with`: `strings.HasSuffix`

Exclude fields use the same semantics.

Normalization:

- lowercase
- trim spaces
- trim trailing dot
- ignore empty values

No DNS-aware suffix behavior exists. For root + subdomains:

```uci
list domain_equals 'example.com'
list domain_ends_with '.example.com'
```

## IPv6 Status

IPv6 routing is not implemented in v1. Do not leak real IPv6 answers for
FakeIP-matched domains while IPv6 routing is absent.
