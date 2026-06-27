# neto

`neto` - policy router для OpenWrt/ImmortalWrt, который принимает routing
решения до попадания трафика в `sing-box`.

**English version:** [README_EN.md](README_EN.md)

Главная идея: `nftables` решает, какой LAN client и какой destination нужно
отправить в proxy, а что должно идти `direct`. `sing-box` не получает весь
трафик подряд и не становится "главным router". Он используется только как:

- владелец `FakeIP` DNS;
- backend для `TProxy` inbound;
- executor для proxy outbounds.

Это принципиально отличается от схемы "весь LAN traffic -> sing-box". В neto
обычный `direct` traffic остается вне sing-box.

## Status

Проект рассчитан на embedded-first установку на OpenWrt/ImmortalWrt.

Поддерживается:

- OpenWrt 23.05+
- OpenWrt 24.10+
- OpenWrt 25.12+
- ImmortalWrt 25.12+
- firewall4 / `nftables`
- IPv4 routing

Не поддерживается в v1:

- IPv6 routing;
- fw3 / iptables;
- custom transparent proxy внутри `netod`;
- custom FakeIP allocator;
- `.ipk` packaging.

## Install

Установка с GitHub:

```sh
sh -c "$(wget -O- https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh)"
```

или:

```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh)"
```

Installer скачивает archive из GitHub Releases:

```text
https://github.com/elllkere/neto/releases/latest/download/neto-openwrt-embedded.tar.gz
```

Если нужен свой mirror:

```sh
NETO_BASE_URL='https://your-host/path' sh -c "$(wget -O- https://raw.githubusercontent.com/elllkere/neto/main/embedded/install.sh)"
```

Для локального archive:

```sh
./embedded/install.sh --local ./dist/neto-openwrt-embedded.tar.gz --verbose
```

## Upgrade

После установки:

```sh
/usr/share/neto/upgrade.sh
```

`upgrade.sh` скачивает свежий installer из GitHub raw. Для своего installer URL:

```sh
NETO_INSTALL_URL='https://your-host/install.sh' /usr/share/neto/upgrade.sh
```

## Quick Check

На router:

```sh
netod check
netod compile
/usr/libexec/neto/sing-box check -c /tmp/neto/sing-box.json
/etc/init.d/neto restart
netod status
```

Проверка DNS:

```sh
dig @127.0.0.1 -p 5353 youtube.com A
dig @127.0.0.1 -p 5353 youtube.com AAAA
```

Проверка nft/TProxy:

```sh
nft list table inet neto
ip -4 rule show
ip -4 route show table 101
```

## Routing Model

`neto` работает только с LAN client traffic.

Порядок в generated `nft`:

1. LAN guard.
2. `direct_clients4` -> `return`.
3. reserved/local destinations -> `return`.
4. `proxy_clients4` -> proxy.
5. `FakeIP` range -> proxy в `custom` mode.
6. Ordered IP/provider/CIDR rules.
7. default `return`.

`routing_mode`:

- `custom`: proxy только по rules/client policy/FakeIP/IP provider.
- `global`: весь non-reserved LAN TCP/UDP идет в proxy, кроме direct clients.

Client policy:

- `default`: следует `routing_mode`;
- `proxy`: весь non-reserved TCP/UDP этого client идет в proxy;
- `direct`: hard bypass, real DNS only, no FakeIP.

Rule action:

- `proxy`: отправить matching traffic в выбранный outbound;
- `direct`: вернуть traffic в обычный routing;
- `block`: block behavior для DNS phase и packet phase через return/block semantics.

## DNS/FakeIP Semantics

DNS chain:

```text
LAN DNS -> dnsmasq -> netod -> selected local sing-box DNS listener
```

Packet chain:

```text
LAN traffic -> nft decides before sing-box
```

`netod` не реализует DoH/DoT/DoQ transport. Он только выбирает DNS path:

- `fakeip`;
- `real-direct`;
- `real-proxy`;
- `block`.

`sing-box` делает actual DNS transport: UDP, TCP, DoT, DoH.

Local DNS listeners:

- FakeIP: `127.0.0.1:15353`
- real-direct: `127.0.0.1:15354`
- real-proxy: `127.0.0.1:15355`

Semantics:

- domain `proxy` rules in `custom` mode use `FakeIP`;
- provider/CIDR/IP rules use real DNS, потому что nft должен видеть real
  destination IP;
- direct rules and direct clients always use real DNS;
- `routing_mode=global` returns real DNS by default;
- AAAA для FakeIP domains возвращает NODATA при включенном
  `filter_aaaa_for_fakeip`.

Mixed rules разрешены:

```uci
config rule
	option action 'proxy'
	list domain_contains 'youtube'
	list ip_provider 'cloudflare_ipv4'
```

Это не AND между domain и IP:

- domain part работает только в DNS phase;
- provider/CIDR/IP part работает только в packet/nft phase;
- оба entry points используют один `action` и один `outbound`.

Port/proto matchers тоже packet-only:

```uci
list proto 'tcp'
list dst_port '443'
```

Ports не применяются к DNS/FakeIP domain matching.

## Outbounds

Built-in outbounds:

- `direct`
- `blocked`

Их не нужно создавать как `config outbound`.

Создаваемые outbound types:

- `vless`
- `hysteria2`
- `shadowsocks`
- `trojan`

Imports/subscriptions создают обычные outbound sections. После import node
можно выбрать в rule outbound dropdown так же, как manual profile.

## Providers

Provider - это data source, а rule - это routing policy.

Provider сам ничего не маршрутизирует. Он начинает влиять на routing только
после ссылки из rule:

```uci
list domain_provider 'youtube_domains'
list ip_provider 'cloudflare_ipv4'
```

Installer добавляет built-in IP providers, если providers с такими URL еще нет:

- Cloudflare IPv4: `https://www.cloudflare.com/ips-v4/`
- Telegram IPv4: `https://core.telegram.org/resources/cidr.txt`

Telegram feed содержит IPv6; neto сохраняет только valid IPv4 CIDR/address
entries.

Manual update:

```sh
netod providers update
netod providers update telegram_ipv4
```

## LuCI

LuCI app находится в `Services -> neto`.

Основные страницы:

- General: service status, DNS settings, routing mode.
- Outbounds: manual profiles, imports, subscriptions.
- Rules: domain/IP/provider rules, packet proto/ports.
- Clients: client policy.
- Providers: remote provider sources.
- Advanced: low-level DNS listeners, dnsmasq, LAN, TProxy, FakeIP range.
- Debug: `netod debug`.

В Rules page `dns_mode` скрыт и пишется как `auto`; DNS behavior выводится из
типа rule автоматически.

## Docs

- [Architecture](docs/ARCHITECTURE.md)
- [Router testing](docs/ROUTER_TESTING.md)
- [Outbounds](docs/OUTBOUNDS.md)
- [Testing](docs/TESTING.md)
- [Decisions](docs/DECISIONS.md)
- [Roadmap](docs/ROADMAP.md)
- [Release process](docs/RELEASE.md)

## Build

Локальная сборка:

```sh
GOCACHE=/tmp/neto-go-cache go test ./...
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
./scripts/test-archive.sh
```

Archive:

```text
dist/neto-openwrt-embedded.tar.gz
```

В archive должен быть top-level directory `neto/`.

## Release Checklist

Чтобы install command из README работал с GitHub без отдельного домена:

1. Собрать archive:

```sh
GOCACHE=/tmp/neto-go-cache ./embedded/pack.sh
./scripts/test-archive.sh
```

2. Создать GitHub Release.
3. Загрузить asset с точным именем:

```text
neto-openwrt-embedded.tar.gz
```

Installer скачивает именно:

```text
https://github.com/elllkere/neto/releases/latest/download/neto-openwrt-embedded.tar.gz
```

Полный процесс: [docs/RELEASE.md](docs/RELEASE.md).
