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
netod rules list
netod test-domain 192.168.8.100 youtube.com A
nft list table inet neto
nft list set inet neto lan_subnets4
ip -4 rule show
ip -4 route show table 101
uci show neto
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
- creating a rule writes `priority`
- moving rules rewrites priority as `100`, `200`, `300`, ...
- Rules page writes only new matcher field names
- Rules page does not write `match_all`
- Rules page does not write deprecated matcher fields
- Providers page does not create rules unless explicitly intended
- Rules page does not create providers

Inspect UCI:

```sh
uci show neto
```

