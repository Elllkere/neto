# Roadmap

This file captures near-term work for future Codex sessions. Do not treat it as
a product promise.

## Current Status

Milestone 1 has working core runtime pieces:

- UCI parsing for main/client/rule config
- ordered rule engine
- IPv4 provider loading
- CIDR normalize/dedup/collapse
- nftables generation
- TProxy policy routing lifecycle
- sing-box config generation
- netod DNS listener
- dnsmasq integration
- native outbound manual profiles
- manual share-link imports and subscription updates for VLESS, Hysteria2,
  Shadowsocks, and Trojan nodes
- OpenWrt/ImmortalWrt embedded installer
- minimal LuCI app

Runtime has been tested on ImmortalWrt 25.12 rockchip/armv8 aarch64 with `apk`.

## Known Current LuCI Bugs / Gaps

Treat these as active until verified on router:

1. LuCI Rules page may delete or not preserve `option enabled`.
2. LuCI-created rules may lack `priority`.
3. Rules table shows too many fields.
4. Providers page may duplicate rules or rule creation may create provider-like
   entries.
5. `match_all` must be removed from v1.
6. LuCI must write only:
   - `domain_equals`
   - `domain_contains`
   - `domain_starts_with`
   - `domain_ends_with`
   - `exclude_domain_equals`
   - `exclude_domain_contains`
   - `exclude_domain_starts_with`
   - `exclude_domain_ends_with`
7. LuCI must not write deprecated fields:
   - `domain_keyword`
   - `domain_suffix`
   - `domain_prefix`
   - `domain_exact`

## Next Planned Tasks

1. Router-verify LuCI Rules page behavior.
2. Fix LuCI `enabled` preservation if still broken.
3. Ensure LuCI-created rules always write explicit `enabled` and `priority`.
4. Separate Providers UX from Rules UX:
   - provider creation must not create rules
   - rule creation must not create providers
5. Add or verify move up/down controls and priority rewriting.
6. Add `netod rules list`.
7. Add `netod test-domain <client-ip> <domain> <qtype>`.
8. Add better rule/debug output for DNS decision traces.
9. Add router-side tests for LuCI-generated UCI.

## Later Tasks

- More robust provider management.
- Optional provider update support.
- More complete LuCI validation and summaries.
- Additional outbound selectors beyond native manual/imported profiles.
- Better block behavior for packet phase.
- IPv6 design, only after v1 IPv4 path is stable.

## Explicitly Not in v1

- IPv6 routing.
- Custom transparent TCP/UDP proxy.
- Custom FakeIP allocator.
- fw3/iptables support.
- Full `.ipk` packaging.
- Advanced multi-outbound UI.
- Automatic provider updates.
- Reintroducing `match_all`.
