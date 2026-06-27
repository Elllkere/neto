'use strict';
'require fs';
'require form';
'require ui';
'require uci';
'require view';
'require neto.i18n as netoI18n';

var _ = netoI18n.translate;

function commandResult(path, args) {
	return fs.exec(path, args).catch(function(err) {
		return {
			code: -1,
			stdout: '',
			stderr: String(err)
		};
	});
}

function outputLine(res) {
	return String((res && (res.stdout || res.stderr)) || '').trim().split('\n')[0] || '-';
}

function processStatus(res) {
	return res && res.code == 0 ? _('Running') : _('Stopped');
}

function serviceStatus(res) {
	return res && res.code == 0 ? _('Running') : _('Stopped');
}

function autostartStatus(res) {
	return res && res.code == 0 ? _('Enabled') : _('Disabled');
}

function defaultDNSPort(protocol) {
	switch (protocol) {
	case 'tls':
		return '853';
	case 'https':
		return '443';
	default:
		return '53';
	}
}

function normalizeDNSProtocol(protocol) {
	switch (String(protocol || '').trim()) {
	case 'tcp':
		return 'tcp';
	case 'tls':
	case 'dot':
		return 'tls';
	case 'https':
	case 'doh':
		return 'https';
	default:
		return 'udp';
	}
}

function normalizeDNSMode(mode) {
	mode = String(mode || '').trim();
	return mode == 'proxy' ? 'proxy' : 'direct';
}

function splitHostPort(value) {
	value = String(value || '').trim();
	var idx = value.lastIndexOf(':');

	if (idx > 0 && value.indexOf(':') == idx)
		return [ value.slice(0, idx), value.slice(idx + 1) ];

	return [ value, '' ];
}

function normalizeDNSState() {
	var mode = normalizeDNSMode(uci.get('neto', 'main', 'real_dns_mode'));
	var protocol = normalizeDNSProtocol(uci.get('neto', 'main', 'real_dns_transport') || uci.get('neto', 'main', 'dns_upstream_protocol'));
	var host = String(uci.get('neto', 'main', 'real_dns_server') || uci.get('neto', 'main', 'dns_upstream_host') || '').trim();
	var port = String(uci.get('neto', 'main', 'real_dns_port') || uci.get('neto', 'main', 'dns_upstream_port') || '').trim();
	var tlsName = String(uci.get('neto', 'main', 'real_dns_server_name') || uci.get('neto', 'main', 'dns_upstream_tls_name') || '').trim();
	var path = String(uci.get('neto', 'main', 'real_dns_path') || uci.get('neto', 'main', 'dns_upstream_path') || '').trim();
	var legacy = splitHostPort(uci.get('neto', 'main', 'real_dns_upstream'));
	var serverParts;

	if (host == '')
		host = legacy[0] || '1.1.1.1';
	serverParts = splitHostPort(host);
	if (serverParts[1] != '') {
		host = serverParts[0];
		if (port == '')
			port = serverParts[1];
	}
	if (port == '')
		port = defaultDNSPort(protocol);
	if (tlsName == '' && host == '1.1.1.1')
		tlsName = 'cloudflare-dns.com';
	if (tlsName == '' && host == '8.8.8.8')
		tlsName = 'dns.google';
	if (protocol == 'https' && path == '')
		path = '/dns-query';

	uci.set('neto', 'main', 'real_dns_mode', mode);
	uci.set('neto', 'main', 'real_dns_transport', protocol);
	uci.set('neto', 'main', 'real_dns_server', host);
	uci.set('neto', 'main', 'real_dns_port', port);
	uci.set('neto', 'main', 'real_dns_server_name', tlsName);
	uci.set('neto', 'main', 'real_dns_path', path);
	uci.set('neto', 'main', 'dns_upstream_preset', 'custom');
	uci.set('neto', 'main', 'dns_upstream_protocol', protocol);
	uci.set('neto', 'main', 'dns_upstream_host', host);
	uci.set('neto', 'main', 'dns_upstream_port', port);
	uci.set('neto', 'main', 'dns_upstream_tls_name', tlsName);
	uci.set('neto', 'main', 'dns_upstream_path', path);
	uci.set('neto', 'main', 'real_dns_upstream', host + ':' + port);
}

function forceGeneralState() {
	uci.set('neto', 'main', 'fakeip_enabled', '1');
	normalizeDNSState();
}

return view.extend({
	load: function() {
		return uci.load('neto').then(function() {
			var singboxBin = uci.get('neto', 'main', 'singbox_bin') || '/usr/libexec/neto/sing-box';

			return Promise.all([
				commandResult('/etc/init.d/neto', [ 'status' ]),
				commandResult('/etc/init.d/neto', [ 'enabled' ]),
				commandResult('/bin/pidof', [ 'netod' ]),
				commandResult('/bin/pidof', [ 'sing-box' ]),
				commandResult('/usr/bin/netod', [ 'version' ]),
				commandResult(singboxBin, [ 'version' ])
			]);
		});
	},

	handleSave: function() {
		return this.map.save(forceGeneralState).then(function() {
			return ui.changes.init();
		});
	},

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

	handleService: function(action) {
		var chain = Promise.resolve();

		if (action == 'start') {
			chain = chain
				.then(function() {
					return fs.exec('/sbin/uci', [ 'set', 'neto.main.enabled=1' ]);
				})
				.then(function(res) {
					if (res.code)
						throw new Error(res.stderr || res.stdout || _('Update failed'));

					return fs.exec('/sbin/uci', [ 'commit', 'neto' ]);
				});
		}

		return chain
			.then(function(res) {
				if (res && res.code)
					throw new Error(res.stderr || res.stdout || _('Update failed'));

				return fs.exec('/etc/init.d/neto', [ action ]);
			})
			.then(function(res) {
				if (res.code)
					throw new Error(res.stderr || res.stdout || _('Update failed'));

				window.location.reload();
			});
	},

	handleAutostart: function(action) {
		return fs.exec('/etc/init.d/neto', [ action ])
			.then(function(res) {
				if (res.code)
					throw new Error(res.stderr || res.stdout || _('Update failed'));

				window.location.reload();
			});
	},

	render: function(data) {
		var m, s, o, state, serviceRunning, autostartEnabled;

		data = data || [];
		state = {
			service: data[0],
			autostart: data[1],
			netod: data[2],
			singbox: data[3],
			netodVersion: data[4],
			singboxVersion: data[5]
		};
		serviceRunning = state.service && state.service.code == 0;
		autostartEnabled = state.autostart && state.autostart.code == 0;

		m = new form.Map('neto', _('neto'));
		this.map = m;

		s = m.section(form.NamedSection, 'main', 'main', _('General'));

		o = s.option(form.DummyValue, '_neto_status', _('neto status'));
		o.cfgvalue = function() {
			return serviceStatus(state.service);
		};

		o = s.option(form.DummyValue, '_singbox_status', _('sing-box status'));
		o.cfgvalue = function() {
			return processStatus(state.singbox);
		};

		o = s.option(form.DummyValue, '_autostart_status', _('Autostart'));
		o.cfgvalue = function() {
			return autostartStatus(state.autostart);
		};

		o = s.option(form.DummyValue, '_netod_version', _('netod version'));
		o.cfgvalue = function() {
			return outputLine(state.netodVersion);
		};

		o = s.option(form.DummyValue, '_singbox_version', _('sing-box version'));
		o.cfgvalue = function() {
			return outputLine(state.singboxVersion);
		};

		o = s.option(form.Button, '_service', _('Service'));
		o.inputtitle = serviceRunning ? _('Stop') : _('Start');
		o.inputstyle = serviceRunning ? 'reset' : 'apply';
		o.onclick = L.bind(function() {
			return this.handleService(serviceRunning ? 'stop' : 'start').catch(function(err) {
				ui.addNotification(null, E('p', {}, [ err.message || err ]), 'danger');
			});
		}, this);

		o = s.option(form.Button, '_autostart', _('Autostart'));
		o.inputtitle = autostartEnabled ? _('Disable') : _('Enable');
		o.inputstyle = autostartEnabled ? 'reset' : 'apply';
		o.onclick = L.bind(function() {
			return this.handleAutostart(autostartEnabled ? 'disable' : 'enable').catch(function(err) {
				ui.addNotification(null, E('p', {}, [ err.message || err ]), 'danger');
			});
		}, this);

		if (netoI18n.ruAvailable()) {
			o = s.option(form.ListValue, 'language', _('Language'));
			o.value('en', _('English'));
			o.value('ru', _('Russian'));
			o.default = 'en';
			o.rmempty = false;
		}

		o = s.option(form.Value, 'dns_listen', _('DNS server'));
		o.placeholder = '127.0.0.1:5353';
		o.rmempty = false;

		o = s.option(form.ListValue, 'real_dns_mode', _('Real DNS mode'));
		o.value('direct', _('Direct'));
		o.value('proxy', _('Proxy'));
		o.default = 'direct';
		o.rmempty = false;

		o = s.option(form.ListValue, 'real_dns_transport', _('Real DNS transport'));
		o.value('udp', _('UDP'));
		o.value('tcp', _('TCP'));
		o.value('tls', _('DNS over TLS'));
		o.value('https', _('DNS over HTTPS'));
		o.default = 'udp';
		o.rmempty = false;

		o = s.option(form.Value, 'real_dns_server', _('Server'));
		o.placeholder = '1.1.1.1';
		o.rmempty = false;

		o = s.option(form.Value, 'real_dns_server_name', _('Server name'));
		o.depends('real_dns_transport', 'tls');
		o.depends('real_dns_transport', 'https');
		o.placeholder = 'cloudflare-dns.com';

		o = s.option(form.Value, 'real_dns_path', _('Path'));
		o.depends('real_dns_transport', 'https');
		o.placeholder = '/dns-query';

		o = s.option(form.Value, 'fakeip_range', _('FakeIP range'));
		o.datatype = 'cidr4';
		o.placeholder = '198.18.0.0/15';

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

		return m.render();
	}
});
