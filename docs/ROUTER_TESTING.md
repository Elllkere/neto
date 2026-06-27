# Router Testing

Run these commands on an OpenWrt or ImmortalWrt 23.05, 24.10, or 25.12+ router
after installing neto.

Routing semantics in v1:

- `routing_mode=custom`: ordered `config rule` sections decide proxy/direct/block; unmatched traffic is `default_outbound=direct`.
- `routing_mode=global`: LAN client non-reserved TCP/UDP traffic is proxied unless the client policy is `direct`.
- Client `policy=default` follows `routing_mode`.
- Client `policy=proxy` forces non-reserved TCP/UDP traffic to `proxy_default`.
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

Provider rules use `list file` with IPv4 CIDR files and compile into nft
interval sets in rule order.

Rules are for explicit domain/IP/provider matches only. To proxy everything
globally, use General -> `routing_mode=global`. To proxy one client entirely,
use client `policy=proxy`.

Example:

```uci
config rule
	option name 'youtube_except_kz'
	option enabled '1'
	option priority '100'
	option action 'proxy'
	option outbound 'proxy_default'
	option dns_mode 'fakeip'
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

Check DNS forwarding through netod:

```sh
dig @127.0.0.1 -p 5353 youtube.com
dig @127.0.0.1 -p 5353 example.org
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
