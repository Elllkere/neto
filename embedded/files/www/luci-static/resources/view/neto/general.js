'use strict';
'require fs';
'require form';
'require uci';
'require view';

return view.extend({
	handleSaveApply: function(ev) {
		return this.handleSave(ev)
			.then(function() {
				return uci.apply();
			})
			.then(function() {
				return fs.exec('/etc/init.d/neto', [ 'restart' ]);
			})
			.then(function() {
				window.location.reload();
			});
	},

	render: function() {
		var m, s, o;

		m = new form.Map('neto', _('neto'));

		s = m.section(form.NamedSection, 'main', 'main', _('General'));

		o = s.option(form.Flag, 'enabled', _('Enabled'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Value, 'dns_listen', _('DNS listen'));
		o.placeholder = '127.0.0.1:5353';

		o = s.option(form.Value, 'real_dns_upstream', _('Real DNS upstream'));
		o.placeholder = '1.1.1.1:53';

		o = s.option(form.Flag, 'manage_dnsmasq', _('Manage dnsmasq'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Flag, 'filter_aaaa_for_fakeip', _('Filter FakeIP AAAA'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.ListValue, 'routing_mode', _('Routing mode'));
		o.value('custom', _('Custom'));
		o.value('global', _('Global'));
		o.default = 'custom';

		o = s.option(form.ListValue, 'default_outbound', _('Default outbound'));
		o.value('direct', _('Direct'));
		o.default = 'direct';

		o = s.option(form.DynamicList, 'lan_subnet', _('LAN IPv4 subnets'));
		o.datatype = 'cidr4';
		o.placeholder = '192.168.8.0/24';

		o = s.option(form.DynamicList, 'lan_iface', _('LAN interfaces'));
		o.placeholder = 'br-lan';

		o = s.option(form.Value, 'singbox_bin', _('sing-box binary'));
		o.placeholder = '/usr/libexec/neto/sing-box';

		o = s.option(form.Value, 'singbox_dns', _('sing-box DNS'));
		o.placeholder = '127.0.0.1:15353';

		o = s.option(form.Value, 'tproxy_port', _('TProxy port'));
		o.datatype = 'port';
		o.placeholder = '16001';

		o = s.option(form.Value, 'mark', _('Mark'));
		o.placeholder = '0x101';

		o = s.option(form.Value, 'table', _('Table'));
		o.datatype = 'uinteger';
		o.placeholder = '101';

		o = s.option(form.Flag, 'fakeip_enabled', _('FakeIP'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Value, 'fakeip_range', _('FakeIP range'));
		o.datatype = 'cidr4';
		o.placeholder = '198.18.0.0/15';

		o = s.option(form.Flag, 'resolve_for_subnet_rules', _('Resolve subnet rules'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;

		o = s.option(form.Flag, 'nft_counters', _('nft counters'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;

		return m.render();
	}
});
