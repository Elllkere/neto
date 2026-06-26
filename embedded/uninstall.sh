#!/bin/sh

set -eu

PURGE=0
if [ "${1:-}" = "--purge" ]; then
	PURGE=1
fi

if [ -x /etc/init.d/neto ]; then
	/etc/init.d/neto stop || true
	/etc/init.d/neto disable || true
fi

dns_listen="$(uci -q get neto.main.dns_listen || echo '127.0.0.1:5353')"
dns_host="${dns_listen%:*}"
dns_port="${dns_listen##*:}"
dns_server="$dns_host#$dns_port"
uci -q del_list dhcp.@dnsmasq[0].server="$dns_server" || true
if [ -f /var/lib/neto/dnsmasq-noresolv.prev ]; then
	old_noresolv="$(cat /var/lib/neto/dnsmasq-noresolv.prev)"
	if [ "$old_noresolv" = "__missing__" ]; then
		uci -q delete dhcp.@dnsmasq[0].noresolv || true
	else
		uci set dhcp.@dnsmasq[0].noresolv="$old_noresolv"
	fi
fi
if [ -f /var/lib/neto/dnsmasq-addsubnet.prev ]; then
	old_addsubnet="$(cat /var/lib/neto/dnsmasq-addsubnet.prev)"
	if [ "$old_addsubnet" = "__missing__" ]; then
		uci -q delete dhcp.@dnsmasq[0].addsubnet || true
	else
		uci set dhcp.@dnsmasq[0].addsubnet="$old_addsubnet"
	fi
fi
uci commit dhcp || true

rm -f /etc/init.d/neto
rm -f /usr/bin/netod
rm -rf /usr/libexec/neto
rm -rf /usr/share/neto
rm -f /usr/share/luci/menu.d/luci-app-neto.json
rm -f /usr/share/rpcd/acl.d/luci-app-neto.json
rm -rf /www/luci-static/resources/view/neto
rm -rf /tmp/neto
rm -rf /var/lib/neto
rm -f /tmp/dnsmasq.d/neto.conf /etc/dnsmasq.d/neto.conf

if [ "$PURGE" -eq 1 ]; then
	rm -f /etc/config/neto
	rm -rf /etc/neto
fi

if [ -x /etc/init.d/rpcd ]; then
	/etc/init.d/rpcd restart || true
fi
if [ -x /etc/init.d/uhttpd ]; then
	/etc/init.d/uhttpd restart || true
fi
if [ -x /etc/init.d/dnsmasq ]; then
	/etc/init.d/dnsmasq reload || /etc/init.d/dnsmasq restart || true
fi

echo "neto uninstalled"
