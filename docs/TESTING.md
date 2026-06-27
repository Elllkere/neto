# Testing

## Local Tests

Run from repository root.

```sh
go test ./...
sh -n embedded/*.sh scripts/*.sh
jq empty embedded/files/usr/share/luci/menu.d/*.json
jq empty embedded/files/usr/share/rpcd/acl.d/*.json
node --check embedded/files/www/luci-static/resources/view/neto/*.js
./embedded/pack.sh
./scripts/test-archive.sh
```

If the sandbox cannot write the default Go cache:

```sh
GOCACHE=/tmp/neto-go-cache go test ./...
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
```

## Archive Expectations

The embedded archive must contain a top-level `neto/` directory:

```sh
tar -tzf dist/neto-openwrt-embedded.tar.gz | sed -n '1,40p'
```

Expected first line:

```text
neto/
```

## Router Tests

Run on OpenWrt/ImmortalWrt after installing.

```sh
/etc/init.d/neto restart
netod check
netod compile
netod apply
netod status
netod debug
netod import-uri -file /tmp/neto-import.txt
netod subscriptions update
netod rules list
netod test-domain 192.168.8.100 youtube.com A
nft list table inet neto
nft list set inet neto lan_subnets4
ip -4 rule show
ip -4 route show table 101
uci show neto
grep -n "neto subscriptions" /etc/crontabs/root
uci show dhcp.@dnsmasq[0] | grep -E "server|noresolv|addsubnet"
logread | tail -n 120
```

`netod rules list` and `netod test-domain` are requested router-debug commands.
If not implemented, keep them on the roadmap.

## DNS Tests on Router

```sh
dig @127.0.0.1 -p 5353 youtube.com A
dig @127.0.0.1 -p 5353 youtube.com AAAA
dig @127.0.0.1 -p 5353 example.org A
dig @127.0.0.1 -p 15353 youtube.com A
```

Expected:

- FakeIP-matched `A` returns `198.18.x.x`.
- FakeIP-matched `AAAA` does not return real IPv6.
- Non-matching domains use real upstream DNS.

## Windows LAN Client Tests

```cmd
ipconfig /flushdns
nslookup -type=A youtube.com 192.168.8.1
nslookup -type=AAAA youtube.com 192.168.8.1
curl -4 -v --connect-timeout 10 https://example.org
curl -4 -v --connect-timeout 10 https://youtube.com
```

Expected:

- direct clients receive real DNS answers only
- FakeIP-matched proxy domains return FakeIP for `A`
- `AAAA` for FakeIP-matched domains does not leak real IPv6
- non-LAN/WAN/inbound traffic is not captured by neto

## nft/TProxy Checks

Check that LAN guard comes before policy decisions:

```sh
nft list table inet neto
```

Expected order:

1. LAN source guard in `prerouting`
2. non-LAN return
3. `direct_clients4` return
4. `reserved4` return
5. `proxy_clients4`
6. FakeIP/provider/rule proxy rules
7. default return

Check TProxy routing:

```sh
ip -4 rule show | grep 'fwmark 0x101'
ip -4 route show table 101
```

Expected route:

```text
local default dev lo
```

On OpenWrt/ImmortalWrt, `ip -4 route show table 101` may exit 2 when the table
does not exist. That state means missing, not fatal.

## LuCI Checks

Manually verify on router:

- creating a rule writes `option enabled '1'`
- disabling a rule writes `option enabled '0'`
- editing a rule preserves `enabled`
- General shows neto/sing-box status and versions
- General Start starts the service and writes `neto.main.enabled=1`
- General Stop stops the service through `/etc/init.d/neto stop`
- General Autostart uses `/etc/init.d/neto enable|disable`
- General exposes only language, routing mode, and default outbound from UCI
- Advanced contains low-level DNS, LAN, sing-box, TProxy, FakeIP range, and nft settings
- LuCI does not expose a FakeIP off switch and Save forces `fakeip_enabled=1`
- creating a rule writes `priority`
- moving rules rewrites priority as `100`, `200`, `300`, ...
- Rules page writes only new matcher field names
- Rules page does not write `match_all`
- Rules page does not write deprecated matcher fields
- Rules page hides DNS mode selection and writes `dns_mode=fakeip` for proxy rules
- Providers page does not create rules unless explicitly intended
- Rules page does not create providers
- Rules page defaults new rule outbound to `direct`
- Rules page offers built-in `direct` and `blocked` outbounds
- Rules page offers enabled custom outbound tags after they are created
- Rules page does not offer `proxy_default`
- Outbounds page creates only VLESS, Hysteria2, Shadowsocks, or Trojan profiles
- Outbounds page creates stable `tag` from the first Add input
- editing an outbound changes only `label`/name, not `tag`
- Outbounds table shows only text name, type, address, and port, without a
  second editable name/input column
- Outbounds page does not show an `enabled` field
- Outbounds page does not expose SOCKS as a primary outbound
- Outbounds page does not expose `direct` as a creatable outbound type
- Outbounds page does not auto-create an empty `proxy_default`
- Outbounds edit dialog exposes homeproxy-like protocol details: VLESS flow as a
  dropdown, Shadowsocks method as a dropdown, TLS min/max/ciphers, allow
  insecure, ECH, uTLS fingerprint, REALITY, and transport-specific fields
- REALITY public key and short ID fields are hidden until `type=vless`,
  `tls=1`, and `reality=1`
- Outbounds page has an Import button next to Add for `vless://`,
  `hysteria2://`, `ss://`, and `trojan://` links
- Outbounds page can save subscription URL, auto-update toggle, update time
  from a `0:00`-`23:00` dropdown, update_via, and optional update outbound
- Outbounds page hides update hour until auto-update is enabled
- Outbounds page Manual Update replaces only nodes for the selected subscription
- Imported nodes appear in the node list and in Rules outbound dropdown
- Subscription nodes appear in the normal Outbounds table and can be edited;
  the next update of that subscription overwrites those nodes again
- Imported nodes carry `option imported '1'`; subscription nodes also carry
  `option subscription '<name>'`
- LuCI Save & Apply restarts neto after committing changes
- General Stop stops neto even if no outbound is configured
- with General Start and no custom outbounds, `netod check` still
  succeeds because `direct` and `blocked` are built-ins

Inspect UCI:

```sh
uci show neto
```
