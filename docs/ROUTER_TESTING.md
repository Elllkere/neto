# Router Testing

Run these commands on an OpenWrt or ImmortalWrt 23.05, 24.10, or 25.12+ router
after installing neto.

Routing semantics in v1:

- `routing_mode=custom`: ordered `config rule` sections decide proxy/direct/block; unmatched traffic is `default_outbound=direct`.
- `routing_mode=global`: LAN client non-reserved TCP/UDP traffic is proxied unless the client policy is `direct`.
- Client `policy=default` follows `routing_mode`.
- Client `policy=proxy` forces non-reserved TCP/UDP traffic through neto.
- Client `policy=direct` bypasses neto and receives real DNS answers.

Rules are evaluated by ascending `priority`; the first rule whose include
conditions match and exclude conditions do not match wins. Domain rule fields are
literal string operations, not DNS-aware matching:

- `domain_equals youtube.com` matches only `youtube.com`.
- `domain_contains youtube` matches `youtube.com`, `youtube.kz`, and `notyoutube.com`.
- `domain_starts_with you` matches `youtube.com`.
- `domain_ends_with youtube.com` matches `youtube.com`, `www.youtube.com`, and `notyoutube.com`.
- `domain_ends_with .youtube.com` matches `www.youtube.com`, but not `youtube.com` or `notyoutube.com`.

For root + subdomains, use both:

```uci
list domain_equals 'youtube.com'
list domain_ends_with '.youtube.com'
```

Exclude fields use the same semantics:
`exclude_domain_equals`, `exclude_domain_contains`,
`exclude_domain_starts_with`, and `exclude_domain_ends_with`.

Provider rules use remote provider caches and compile IP providers into nft
interval sets in rule order. Rules can also use inline IPv4 addresses/CIDRs:

```uci
list ip_cidr '1.1.1.1'
list ip_cidr '8.8.8.0/24'
```

Remote providers are configured like this:

```uci
config provider 'youtube_domains'
	option enabled '1'
	option label 'YouTube domains'
	option type 'domain'
	option url 'https://example.com/youtube-domains.txt'
	option auto_update '1'
	option update_hour '3'
	option update_via 'direct'

config provider 'google_ips'
	option enabled '1'
	option label 'Google IPs'
	option type 'ip'
	option url 'https://example.com/google-cidrs.txt'
```

Update providers manually:

```sh
netod providers update
netod providers update youtube_domains
/etc/init.d/neto restart
```

Use LuCI Rules -> Domain input / IP input to choose between field lists,
textboxes, and providers:

```uci
list domain_provider 'youtube_domains'
list ip_provider 'google_ips'
```

Provider domain lists are exact-domain lists, one domain per line, with `#`
comments. Provider IP lists contain IPv4 addresses or CIDRs; single IPv4
addresses become `/32`.

Rules default to built-in `option outbound 'direct'`. Built-in `blocked` is also
available. After adding a custom native sing-box outbound profile, select its
tag in the rule outbound field. In LuCI, the first Add input becomes the stable
tag and later edits change only the `label`. See `docs/OUTBOUNDS.md` for VLESS
+ REALITY, Hysteria2, Shadowsocks 2022, and Trojan TLS examples.

Imported nodes are normal outbound sections too. After importing a share link or
running a subscription update, select the imported node tag in the rule outbound
field:

```sh
netod import-uri -file /tmp/neto-import.txt
netod subscriptions update my_sub
netod providers update
/etc/init.d/neto restart
uci show neto | grep "=outbound"
```

Rules are for explicit domain/IP/provider matches only. To proxy everything
globally, use General -> `routing_mode=global`. To proxy one client entirely,
use client `policy=proxy`.

LuCI hides DNS mode and writes `dns_mode=auto`. netod derives DNS behavior:

- domain proxy rules in custom mode use FakeIP
- provider/CIDR proxy rules use real DNS so nftables can match real
  destination IPs
- direct rules and direct clients always use real DNS
- global mode returns real DNS by default because nftables already enforces the
  global proxy path

Do not mix domain and provider/CIDR matchers in one rule. netod check rejects
mixed rules; split them into separate rules.

Example:

```uci
config rule
	option name 'youtube_except_kz'
	option enabled '1'
	option priority '100'
	option action 'proxy'
	option outbound 'my_vless'
	option dns_mode 'auto'
	list domain_contains 'youtube'
	list exclude_domain_equals 'youtube.kz'
	list exclude_domain_ends_with '.youtube.kz'
```

Neto only routes LAN client traffic. Configure at least one `list lan_subnet`
in `/etc/config/neto`, for example `192.168.8.0/24`. Optional
`list lan_iface 'br-lan'` entries further restrict capture by ingress
interface. WAN inbound traffic, router self traffic, and non-LAN prerouting
traffic must return before any proxy/tproxy rule.

```sh
netod check
netod compile
netod apply
netod debug
```

Inspect the generated nftables table and TProxy policy routing:

```sh
nft list table inet neto
nft list set inet neto lan_subnets4
ip -4 rule show
ip -4 route show table 101
```

Validate sing-box compatibility with the generated config:

```sh
sing-box check -c /tmp/neto/sing-box.json
```

If `/etc/config/neto` uses the managed binary, run:

```sh
/usr/libexec/neto/sing-box check -c /tmp/neto/sing-box.json
```

For FakeIP-based proxying, `/tmp/neto/sing-box.json` should contain
`"store_fakeip": true` under `experimental.cache_file`, and `route.final`
should be the selected custom outbound tag.

Check DNS forwarding through netod:

```sh
dig @127.0.0.1 -p 5353 youtube.com
dig @127.0.0.1 -p 5353 example.org
```

DNS terminology:

- `dns_listen` / General -> DNS server is the local netod listener used by
  dnsmasq.
- netod forwards DNS wire queries to local sing-box DNS listeners:
  `127.0.0.1:15353` for FakeIP, `127.0.0.1:15354` for real-direct, and
  `127.0.0.1:15355` for real-proxy.
- `real_dns_mode` selects direct or proxy real-DNS forwarding.
- `real_dns_transport` selects UDP, TCP, DoT, or DoH. sing-box handles that
  transport; netod does not implement encrypted DNS clients.

Example DoH upstream:

```uci
option real_dns_mode 'direct'
option real_dns_transport 'https'
option real_dns_server '1.1.1.1'
option real_dns_port '443'
option real_dns_server_name 'cloudflare-dns.com'
option real_dns_path '/dns-query'
```

LAN client DNS test from Windows:

```cmd
ipconfig /flushdns
nslookup -type=A youtube.com 192.168.8.1
nslookup -type=AAAA youtube.com 192.168.8.1
```

Expected:

- `A` for a FakeIP domain returns `198.18.x.x`.
- `AAAA` for a FakeIP domain returns no real IPv6 answer.
- `example.org` should go through the configured real DNS upstream.

Useful lifecycle checks:

```sh
/etc/init.d/neto restart
netod status
nft list table inet neto
ip rule show | grep 'fwmark 0x101'
ip -4 route show table 101
uci show dhcp.@dnsmasq[0] | grep -E "server|noresolv|addsubnet"
```

Stopping neto should remove only neto-owned state:

```sh
/etc/init.d/neto stop
nft list table inet neto
ip rule show | grep 'fwmark 0x101'
ip -4 route show table 101
uci show dhcp.@dnsmasq[0] | grep '127.0.0.1#5353'
```

When `manage_dnsmasq=1`, neto also sets dnsmasq `addsubnet=32` while running
so netod can see the original LAN client IPv4 through EDNS Client Subnet and
apply `policy=direct` DNS bypass correctly. Stop/uninstall restores the previous
dnsmasq `addsubnet` and `noresolv` values.

Local archive testing on a router:

```sh
./embedded/install.sh --local ./dist/neto-openwrt-embedded.tar.gz --verbose
```

Dry-run argument validation:

```sh
./embedded/install.sh --dry-run
./embedded/install.sh --local ./dist/neto-openwrt-embedded.tar.gz --dry-run
```
