# Decisions

This file records project decisions that should not be casually reversed.

## D1: neto Routes Before sing-box

Routing decisions happen in nftables before traffic reaches sing-box.

Consequences:

- direct/bypass traffic must never enter sing-box
- sing-box is backend only
- nft rules are the data-plane policy authority

## D2: sing-box Owns FakeIP

neto does not implement a FakeIP allocator in v1. sing-box owns FakeIP and the
FakeIP-to-domain mapping.

Consequences:

- FakeIP DNS queries must go to sing-box
- netod may forward DNS but must not synthesize FakeIP addresses itself

## D3: netod Is Not a Transparent Proxy

netod must not handle transparent TCP/UDP data-plane traffic.

Consequences:

- TProxy target is sing-box
- netod can manage config, DNS forwarding, nft, and status
- netod must not become a proxy engine

## D4: nftables/firewall4 Only

Only nftables/firewall4 is supported. fw3/iptables support is forbidden for this
project.

## D5: LAN-Only Capture

neto must only route LAN client traffic. WAN, inbound, router self, and non-LAN
prerouting traffic must return before any proxy rule.

Consequences:

- generated nft must include a LAN source guard
- global mode must still be scoped to LAN sources

## D6: Embedded-First Distribution

The primary distribution is an embedded archive and installer, not an `.ipk`.
The installer must auto-detect architecture and never require users to choose a
binary manually.

## D7: Managed sing-box Never Overwrites System sing-box

If system sing-box is missing or incompatible, managed sing-box lives at:

```text
/usr/libexec/neto/sing-box
```

Never overwrite `/usr/bin/sing-box`.

## D8: Provider Is Data, Rule Is Policy

Provider files are data sources. Rules are routing policy.

Consequences:

- creating a provider must not create a rule
- creating a rule must not create a provider
- provider CIDRs are compiled into nft sets referenced by rules

## D9: Ordered Rules, First Match Wins

Rules are evaluated by ascending `priority`. The first terminal matching rule
wins.

Rule matching requires:

- include condition matched
- exclude condition did not match

Rules with no include conditions do not match.

## D10: Domain Matchers Are Literal String Operations

Domain matchers are not DNS-aware.

- `domain_equals`: `==`
- `domain_contains`: `strings.Contains`
- `domain_starts_with`: `strings.HasPrefix`
- `domain_ends_with`: `strings.HasSuffix`

For root + subdomains:

```uci
list domain_equals 'example.com'
list domain_ends_with '.example.com'
```

## D11: match_all Removed From v1

`match_all` is removed from v1 because it is confusing, duplicates global mode,
and can accidentally capture too much traffic.

Use:

- `routing_mode=global` to proxy everything globally
- client `policy=proxy` to proxy one client entirely

Rules are for explicit domain/IP/provider matches only.

## D12: IPv6 Not Implemented in v1

IPv6 routing is out of scope for v1.

Consequence:

- AAAA for FakeIP-matched domains must not leak real IPv6 while IPv6 routing is
  absent

