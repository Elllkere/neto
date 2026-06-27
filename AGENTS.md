# neto Agent Handoff

This repository contains `neto`, an embedded-first OpenWrt/ImmortalWrt service.
Future Codex sessions should read this file before making changes.

## Project Goal

`neto` is a pre-sing-box policy router for OpenWrt/ImmortalWrt. Routing
decisions must happen before traffic enters sing-box. Traffic that should be
direct/bypassed must never enter sing-box.

sing-box is used only as:

- FakeIP DNS owner.
- TProxy inbound backend.
- Proxy outbound executor.

`netod` must not become a transparent TCP/UDP proxy.

## Supported Targets

- OpenWrt 23.05
- OpenWrt 24.10
- OpenWrt 25.12+
- ImmortalWrt 25.12+ has been tested in development.

Only firewall4/nftables is supported. Do not add fw3/iptables support.

## Install Model

The project is embedded-first. Users should install without manually choosing
CPU architecture or binaries:

```sh
sh -c "$(wget -O- https://example.com/neto/install.sh)"
sh -c "$(curl -fsSL https://example.com/neto/install.sh)"
```

The embedded archive is `dist/neto-openwrt-embedded.tar.gz` and should contain a
top-level `neto/` directory.

## Runtime Paths

- `/etc/config/neto`
- `/etc/init.d/neto`
- `/usr/bin/netod`
- `/usr/libexec/neto/sing-box`
- `/usr/share/neto/`
- `/tmp/neto/neto.nft`
- `/tmp/neto/sing-box.json`
- `/var/lib/neto/`

## Critical Invariants

- neto routes before sing-box.
- nftables decides packet routing before sing-box.
- direct/bypass traffic must never enter sing-box.
- sing-box owns FakeIP and the FakeIP domain mapping.
- netod must not proxy transparent TCP/UDP traffic.
- neto must only route LAN client traffic.
- WAN, inbound, router self, and non-LAN prerouting traffic must return.
- IPv6 routing is not implemented in v1.
- AAAA for FakeIP-matched domains must not leak real IPv6 while IPv6 routing is
  absent.
- Large CIDR lists must compile into nft sets, not thousands of nft rules.
- Provider is a data source only.
- Rule is routing policy only.
- Creating a rule must not create a provider.
- Creating a provider must not create a rule.
- `match_all` is removed from v1.

## Current Routing Semantics

Client policies:

- absent/default: follow general `routing_mode`.
- `proxy`: force non-reserved TCP/UDP from this client through neto.
- `direct`: hard bypass. Real DNS only, no FakeIP, nft return before proxy rules.

Routing modes:

- `custom`: selective routing by ordered rules. Unmatched traffic follows
  `default_outbound`, currently only `direct`.
- `global`: all LAN client TCP/UDP goes to proxy except direct clients and
  reserved/local destinations.

## Current DNS Model

- `dns_listen` is the local netod DNS server/listener used by dnsmasq.
- `real_dns_upstream` is a legacy compatibility `host:port` mirror of the real
  upstream.
- New configs should use `dns_upstream_preset`, `dns_upstream_protocol`,
  `dns_upstream_host`, `dns_upstream_port`, `dns_upstream_tls_name`, and
  `dns_upstream_path`.
- Supported real upstream transports are `udp`, `tcp`, `tls` (DoT), and
  `https` (DoH).
- Built-in DNS presets are `cloudflare` and `google`; `custom` uses explicit
  host/port/SNI/path fields.
- The same effective real DNS upstream must be used by netod DNS forwarding and
  generated sing-box DNS server `local`.
- FakeIP DNS decisions still route to sing-box FakeIP DNS; direct/real DNS
  decisions use the real upstream.

## Rule Matcher Semantics

Domain matchers are literal string operations, not DNS-aware matching.

- `domain_equals`: `normalizedDomain == normalizedValue`
- `domain_contains`: `strings.Contains(normalizedDomain, normalizedValue)`
- `domain_starts_with`: `strings.HasPrefix(normalizedDomain, normalizedValue)`
- `domain_ends_with`: `strings.HasSuffix(normalizedDomain, normalizedValue)`

Exclude fields use the same semantics:

- `exclude_domain_equals`
- `exclude_domain_contains`
- `exclude_domain_starts_with`
- `exclude_domain_ends_with`

Normalization:

- lowercase
- trim spaces
- trim trailing dot
- ignore empty values

For root + subdomains use both:

```uci
list domain_equals 'example.com'
list domain_ends_with '.example.com'
```

LuCI must not write deprecated matcher names:

- `domain_keyword`
- `domain_suffix`
- `domain_prefix`
- `domain_exact`

Rules can be filled from multiple LuCI input modes without changing matcher
semantics:

- domain fields: current `domain_*` and `exclude_domain_*` list fields
- domain textbox: writes the same `domain_*` and `exclude_domain_*` UCI lists
- domain files: `list domain_file`, one exact domain per line, comments with `#`
- IP/CIDR list/textbox: `list ip_cidr`, IPv4 addresses become `/32`
- IP/CIDR files: `list ip_file`; legacy `list file` is accepted as an alias

## Current Outbound Model

- Built-in outbound tags are `direct` and `blocked`.
- Built-ins are generated for sing-box and must not be created as
  `config outbound` sections.
- Creatable outbound types are `vless`, `hysteria2`, `shadowsocks`, and
  `trojan`.
- Custom outbounds use stable UCI section/tag IDs plus editable `label`.
- Custom outbounds do not have an enable/disable switch in v1.
- Outbounds LuCI should keep the table compact: text label/name section title,
  type, address, and port only. Do not add a second editable name/input column.
  Protocol details belong in the edit modal.
- Outbounds LuCI should expose homeproxy-like controls for supported protocols:
  VLESS flow dropdown, Shadowsocks method dropdown, TLS min/max/ciphers, allow
  insecure, ECH, uTLS fingerprint, REALITY, and V2Ray transport fields. Hide
  REALITY public key/short ID until REALITY is enabled.
- `proxy_default` is deprecated. LuCI must not create or offer it.
- Old rules with `option outbound 'proxy_default'` may be normalized to
  `direct` for compatibility.

## Current Import / Subscription Model

- Manual imports and subscriptions create normal `config outbound` sections.
- Imported nodes are selectable by rules through the same outbound tag dropdown
  as manual profiles.
- LuCI import and subscription management lives on the Outbounds page. Do not
  add a separate Imports tab unless the user explicitly asks for it.
- Imported outbound sections should carry `option imported '1'`; subscription
  nodes also carry `option subscription '<subscription_name>'`.
- Subscription nodes are ordinary editable outbounds in the main Outbounds
  table. A later update of the same subscription overwrites those nodes again.
- `config subscription '<name>'` supports `enabled`, `label`, `url`,
  `auto_update`, `update_hour`, `update_via`, and `update_outbound`.
- Supported import URI schemes are `vless://`, `hysteria2://`/`hy2://`,
  `ss://`, and `trojan://`.
- `netod import-uri -file <path>` imports one or more share links from a local
  file.
- `netod subscriptions update [name]` downloads subscriptions and replaces only
  nodes belonging to that subscription.
- Subscription downloads use the system `curl` binary, not Go `net/http`, to
  keep embedded multi-architecture `netod` binaries small.
- `auto_update=1` is implemented by neto-owned cron entries in
  `/etc/crontabs/root`; preserve user cron lines and only rewrite the marked
  neto block.
- `update_via=direct` uses direct curl fetching. `update_via=proxy` uses a
  temporary sing-box mixed inbound and a selected custom outbound; it must not
  route router-self traffic through nftables.
- Subscription update intervals are stored in UCI for scheduling/UX; manual
  update is currently the explicit LuCI action.

Do not assume a LuCI issue is fixed until it has been checked on an actual
OpenWrt/ImmortalWrt LuCI instance.

## Forbidden Changes

- Do not add IPv6 routing in v1.
- Do not implement a transparent TCP/UDP proxy in netod.
- Do not implement a custom FakeIP allocator in v1.
- Do not add fw3/iptables support.
- Do not route WAN/inbound/non-LAN prerouting traffic.
- Do not overwrite `/usr/bin/sing-box`.
- Do not make providers create rules or rules create providers.
- Do not generate thousands of nft rules for CIDR lists.
- Do not reintroduce `match_all`.

## Local Commands

```sh
go test ./...
sh -n embedded/*.sh scripts/*.sh
jq empty embedded/files/usr/share/luci/menu.d/*.json
jq empty embedded/files/usr/share/rpcd/acl.d/*.json
node --check embedded/files/www/luci-static/resources/view/neto/*.js
./embedded/pack.sh
./scripts/test-archive.sh
```

If Go cache under `$HOME` is read-only in the sandbox, use:

```sh
GOCACHE=/tmp/neto-go-cache go test ./...
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
```

## Documentation Lookup

When changing LuCI JavaScript under
`embedded/files/www/luci-static/resources/view/neto/`, query the current
OpenWrt LuCI documentation with Context7 before relying on `form` or `uci` API
details. Verify names and behavior for `form.Map`, sections, option flags,
save hooks, and `uci.sections`/`uci.set` persistence.

## Distribution Rebuild

After making repository changes, rebuild the embedded distribution archive and
validate it:

```sh
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
./scripts/test-archive.sh
```
