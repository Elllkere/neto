# Testing

Документ описывает local checks и router checks для `neto`.

## Local Tests

Запускать из repository root:

```sh
go test ./...
sh -n embedded/*.sh scripts/*.sh
jq empty embedded/files/usr/share/luci/menu.d/*.json
jq empty embedded/files/usr/share/rpcd/acl.d/*.json
node --check embedded/files/www/luci-static/resources/view/neto/*.js
./embedded/pack.sh
./scripts/test-archive.sh
```

Если sandbox не может писать в default Go cache:

```sh
GOCACHE=/tmp/neto-go-cache go test ./...
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
```

## Archive Checks

Embedded archive:

```text
dist/neto-openwrt-embedded.tar.gz
```

Проверить layout:

```sh
tar -tzf dist/neto-openwrt-embedded.tar.gz | sed -n '1,40p'
```

Expected first line:

```text
neto/
```

Automated archive check:

```sh
./scripts/test-archive.sh
```

## Router Smoke Test

На OpenWrt/ImmortalWrt router:

```sh
/etc/init.d/neto restart
netod check
netod compile
netod status
netod debug
nft list table inet neto
ip -4 rule show
ip -4 route show table 101
```

Validate sing-box config:

```sh
/usr/libexec/neto/sing-box check -c /tmp/neto/sing-box.json
```

or, if system sing-box is used:

```sh
sing-box check -c /tmp/neto/sing-box.json
```

## DNS Tests

Local netod listener:

```sh
dig @127.0.0.1 -p 5353 youtube.com A
dig @127.0.0.1 -p 5353 youtube.com AAAA
dig @127.0.0.1 -p 5353 example.org A
```

sing-box DNS listeners:

```sh
dig @127.0.0.1 -p 15353 youtube.com A
dig @127.0.0.1 -p 15354 example.org A
dig @127.0.0.1 -p 15355 example.org A
```

Expected:

- FakeIP matched `A` returns `198.18.x.x`.
- FakeIP matched `AAAA` does not return real IPv6.
- Non-matching domains use real DNS.
- Direct clients get real DNS only.

## Windows LAN Client Tests

```cmd
ipconfig /flushdns
nslookup -type=A youtube.com 192.168.8.1
nslookup -type=AAAA youtube.com 192.168.8.1
curl -4 -v --connect-timeout 10 https://example.org
curl -4 -v --connect-timeout 10 https://youtube.com
```

Expected:

- direct clients receive real DNS answers only;
- FakeIP domains return FakeIP for `A`;
- FakeIP domains do not leak real IPv6 via `AAAA`;
- non-LAN/WAN/inbound/router-self traffic is not captured.

## nft/TProxy Checks

Check rule order:

```sh
nft list table inet neto
```

Expected order:

1. LAN source guard in `prerouting`.
2. non-LAN return.
3. `direct_clients4` return.
4. `reserved4` return.
5. `proxy_clients4`.
6. FakeIP/provider/rule proxy rules.
7. default return.

Check TProxy route:

```sh
ip -4 rule show | grep 'fwmark 0x101'
ip -4 route show table 101
```

Expected route:

```text
local default dev lo
```

On OpenWrt/ImmortalWrt, `ip -4 route show table 101` may exit with code 2 when
table does not exist. That means missing state, not fatal command failure.

## Provider Checks

First use the LuCI Providers page `Import provider presets` action when checking
built-in provider names on a fresh install.

```sh
netod providers update
netod providers update cloudflare_ipv4
netod providers update telegram_ipv4
ls -la /etc/neto/provider-cache/
```

Telegram provider must save only IPv4 entries. IPv6 lines from the Telegram feed
are ignored.

## Import/Subscription Checks

Manual import:

```sh
netod import-uri -file /tmp/neto-import.txt
uci show neto | grep '=outbound'
```

Subscription update:

```sh
netod subscriptions update my_sub
uci show neto | grep "subscription='my_sub'"
```

Cron block:

```sh
grep -n "neto subscriptions" /etc/crontabs/root
grep -n "netod providers update" /etc/crontabs/root
```

## LuCI Manual Checks

Verify on actual OpenWrt/ImmortalWrt LuCI:

- General shows neto/sing-box status and versions.
- General Start/Stop calls `/etc/init.d/neto start|stop`.
- General Autostart calls `/etc/init.d/neto enable|disable`.
- General exposes DNS preset/mode/outbound/transport and routing mode.
- Advanced contains DNS listener, FakeIP range, AAAA filter, dnsmasq, LAN,
  sing-box listener, TProxy and nft settings.
- Rules page writes explicit `enabled` and `priority`.
- Moving rules rewrites priority as `100`, `200`, `300`, ...
- Rules page writes only current matcher fields, not deprecated aliases.
- Rules page hides `dns_mode` and writes `dns_mode=auto`.
- Domain input supports fields, textbox and domain providers.
- IP input supports inline IP/CIDR list, textbox and IP providers.
- Rules details include packet-only Protocol, Source ports and Destination
  ports.
- Ports are not shown in main rules table.
- Rules page does not create providers.
- Providers page does not create rules.
- Outbounds page creates only VLESS, Hysteria2, Shadowsocks or Trojan profiles.
- Outbounds page does not create `proxy_default`.
- Imported/subscription nodes appear in Outbounds and in Rules outbound
  dropdown.

Inspect UCI after LuCI save:

```sh
uci show neto
```

## Debug Commands

```sh
netod debug
logread | grep -E 'netod|sing-box|dnsmasq' | tail -n 120
grep -nE 'rule_set|rule-set|/tmp/sing-box/rulesets|"detour": "direct"' /tmp/neto/sing-box.json
```

The grep check should not find legacy sing-box rule-set paths or
`"detour": "direct"`.
