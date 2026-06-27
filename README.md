# neto

neto is an OpenWrt/ImmortalWrt-first pre-sing-box policy router. Routing
decisions are made with firewall4/nftables before traffic reaches sing-box;
sing-box is used only as the FakeIP DNS owner, TProxy inbound backend, and proxy
outbound executor.

Milestone 1 provides UCI parsing, ordered rules with include/exclude domain
matching, native sing-box outbound profiles, share-link/subscription imports,
IPv4 provider loading, nftables and sing-box config generation,
OpenWrt/ImmortalWrt init integration, embedded installer scripts, and a LuCI
application.
