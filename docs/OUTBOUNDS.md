# Outbound Profiles

`direct` and `blocked` are built-in outbound tags. They are always generated for
sing-box and must not be created as `config outbound` sections.

The Outbounds LuCI page creates only real proxy profiles:

- `vless`
- `hysteria2`
- `shadowsocks`
- `trojan`

The LuCI editor keeps the table compact (`label`, `type`, `address`, `port`) and
shows protocol-specific details in the edit dialog. VLESS flow, Shadowsocks
method, TLS versions, ECH, uTLS, REALITY, and V2Ray transport fields are
dropdown/list controls where possible. REALITY public key/short ID are shown
only when REALITY is enabled.

Rules default to `option outbound 'direct'`. After adding a custom outbound,
select it in the rule outbound dropdown. The first Add input becomes the stable
UCI section/tag; later edits should change `label`, not the tag.

`proxy_default` is deprecated. Old rules using it are treated as `direct`, and
old `proxy_default` outbound sections are ignored.

The parser also accepts these homeproxy-style aliases for compatibility:
`address`, `vless_flow`, `tls_sni`, `tls_alpn`, `tls_insecure`, `tls_ech`,
`tls_ech_config`, `tls_ech_config_path`, `tls_utls`, `tls_reality`,
`tls_reality_public_key`, `tls_reality_short_id`, `grpc_servicename`, and
`shadowsocks_encrypt_method`.

## VLESS + REALITY

```uci
config outbound 'my_vless'
	option tag 'my_vless'
	option label 'My VLESS'
	option type 'vless'
	option server 'example.com'
	option port '443'
	option uuid 'a3482e88-686a-4a58-8126-99c9df64b060'
	option flow 'xtls-rprx-vision'
	option tls '1'
	option server_name 'www.example.com'
	option reality '1'
	option reality_public_key 'PUBLIC_KEY'
	option reality_short_id '0123456789abcdef'
	option alpn 'h2,http/1.1'
	option tls_min_version '1.2'
	option tls_max_version '1.3'
	option utls_fingerprint 'chrome'
	option transport 'tcp'
	option packet_encoding 'xudp'
```

## Hysteria2

```uci
config outbound 'my_hy2'
	option tag 'my_hy2'
	option label 'My Hysteria2'
	option type 'hysteria2'
	option server 'example.com'
	option port '443'
	option password 'PASSWORD'
	option server_name 'example.com'
	option insecure '0'
	option hysteria_obfs_type 'salamander'
	option hysteria_obfs_password 'OBFS_PASSWORD'
	option hysteria_down_mbps '100'
	option hysteria_up_mbps '20'
```

## Shadowsocks 2022

```uci
config outbound 'my_ss'
	option tag 'my_ss'
	option label 'My Shadowsocks'
	option type 'shadowsocks'
	option server 'example.com'
	option port '8388'
	option method '2022-blake3-aes-128-gcm'
	option password 'PASSWORD'
```

## Trojan TLS

```uci
config outbound 'my_trojan'
	option tag 'my_trojan'
	option label 'My Trojan'
	option type 'trojan'
	option server 'example.com'
	option port '443'
	option password 'PASSWORD'
	option tls '1'
	option server_name 'example.com'
	option insecure '0'
	option tls_min_version '1.2'
	option tls_max_version '1.3'
	option transport 'ws'
	option ws_host 'example.com'
	option ws_path '/ws'
	option websocket_early_data '2048'
	option websocket_early_data_header 'Sec-WebSocket-Protocol'
```

Subscriptions and imports are not implemented in this milestone.
