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

function isReservedOutboundTag(tag) {
	return tag == 'direct' || tag == 'blocked' || tag == 'block' || tag == 'proxy_default';
}

function proxyOutboundExists(tag) {
	var found = false;

	tag = String(tag || '').trim();
	if (tag == '' || isReservedOutboundTag(tag))
		return false;

	uci.sections('neto', 'outbound', function(section, sid) {
		var existing = String(section.tag || sid || section['.name'] || '').trim();

		if (existing == tag)
			found = true;
	});

	return found;
}

function addProxyOutboundChoices(option) {
	option.value('', _('Select outbound'));

	uci.sections('neto', 'outbound', function(section, sid) {
		var tag = String(section.tag || sid || section['.name'] || '').trim();
		var label = String(section.label || section.name || tag).trim();

		if (tag == '' || isReservedOutboundTag(tag))
			return;

		option.value(tag, label || tag);
	});
}

function normalizeDNSPreset(preset) {
	preset = String(preset || '').trim();
	if (preset == 'google' || preset == 'custom')
		return preset;
	return 'cloudflare';
}

function splitHostPort(value) {
	value = String(value || '').trim();
	var idx = value.lastIndexOf(':');

	if (idx > 0 && value.indexOf(':') == idx)
		return [ value.slice(0, idx), value.slice(idx + 1) ];

	return [ value, '' ];
}

function presetDNS(preset, protocol) {
	switch (preset) {
	case 'google':
		if (protocol == 'tls' || protocol == 'https') {
			return {
				server: 'dns.google',
				serverName: 'dns.google',
				path: '/dns-query'
			};
		}
		return {
			server: '8.8.8.8',
			serverName: 'dns.google',
			path: '/dns-query'
		};
	default:
		return {
			server: '1.1.1.1',
			serverName: 'cloudflare-dns.com',
			path: '/dns-query'
		};
	}
}

function splitDoHValue(value, fallbackName, fallbackPath) {
	value = String(value || '').trim().replace(/^https:\/\//, '');
	fallbackName = String(fallbackName || '').trim();
	fallbackPath = String(fallbackPath || '').trim() || '/dns-query';

	if (value == '')
		return [ fallbackName, fallbackPath ];

	var slash = value.indexOf('/');
	if (slash < 0)
		return [ value, fallbackPath ];

	var name = value.slice(0, slash).trim();
	var path = value.slice(slash).trim();
	return [ name || fallbackName, path || fallbackPath ];
}

function normalizeDNSState() {
	var preset = normalizeDNSPreset(uci.get('neto', 'main', 'dns_upstream_preset'));
	var mode = normalizeDNSMode(uci.get('neto', 'main', 'real_dns_mode'));
	var dnsOutbound = String(uci.get('neto', 'main', 'real_dns_outbound') || '').trim();
	var protocol = normalizeDNSProtocol(uci.get('neto', 'main', 'real_dns_transport') || uci.get('neto', 'main', 'dns_upstream_protocol'));
	var host = String(uci.get('neto', 'main', 'real_dns_server') || uci.get('neto', 'main', 'dns_upstream_host') || '').trim();
	var port = '';
	var tlsName = String(uci.get('neto', 'main', 'real_dns_server_name') || uci.get('neto', 'main', 'dns_upstream_tls_name') || '').trim();
	var path = String(uci.get('neto', 'main', 'real_dns_path') || uci.get('neto', 'main', 'dns_upstream_path') || '').trim();
	var legacy = splitHostPort(uci.get('neto', 'main', 'real_dns_upstream'));
	var serverParts, presetValues, dohParts;

	if (preset != 'custom') {
		presetValues = presetDNS(preset, protocol);
		host = presetValues.server;
		tlsName = presetValues.serverName;
		path = presetValues.path;
		port = defaultDNSPort(protocol);
	} else if (host == '') {
		host = legacy[0] || '1.1.1.1';
		port = legacy[1] || '';
	}

	serverParts = splitHostPort(host);
	if (serverParts[1] != '') {
		host = serverParts[0];
		port = serverParts[1];
	}
	if (port == '')
		port = defaultDNSPort(protocol);
	if (tlsName == '' && host == '1.1.1.1')
		tlsName = 'cloudflare-dns.com';
	if (tlsName == '' && host == '8.8.8.8')
		tlsName = 'dns.google';
	if (protocol == 'https' && preset == 'custom') {
		dohParts = splitDoHValue(uci.get('neto', 'main', '_real_dns_doh'), tlsName, path);
		tlsName = dohParts[0];
		path = dohParts[1];
	}
	if (path == '')
		path = '/dns-query';
	if (dnsOutbound == '' || isReservedOutboundTag(dnsOutbound) || !proxyOutboundExists(dnsOutbound))
		uci.unset('neto', 'main', 'real_dns_outbound');
	else
		uci.set('neto', 'main', 'real_dns_outbound', dnsOutbound);

	uci.set('neto', 'main', 'real_dns_mode', mode);
	uci.set('neto', 'main', 'real_dns_transport', protocol);
	uci.set('neto', 'main', 'real_dns_server', host + ':' + port);
	uci.set('neto', 'main', 'real_dns_port', port);
	uci.set('neto', 'main', 'real_dns_server_name', tlsName);
	uci.set('neto', 'main', 'real_dns_path', path);
	uci.set('neto', 'main', 'dns_upstream_preset', preset);
	uci.set('neto', 'main', 'dns_upstream_protocol', protocol);
	uci.set('neto', 'main', 'dns_upstream_host', host);
	uci.set('neto', 'main', 'dns_upstream_port', port);
	uci.set('neto', 'main', 'dns_upstream_tls_name', tlsName);
	uci.set('neto', 'main', 'dns_upstream_path', path);
	uci.set('neto', 'main', 'real_dns_upstream', host + ':' + port);
	uci.unset('neto', 'main', '_real_dns_doh');
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

		o = s.option(form.ListValue, 'dns_upstream_preset', _('DNS'));
		o.value('cloudflare', _('Cloudflare'));
		o.value('google', _('Google'));
		o.value('custom', _('Custom'));
		o.default = 'cloudflare';
		o.rmempty = false;

		o = s.option(form.ListValue, 'real_dns_mode', _('DNS mode'));
		o.value('direct', 'direct');
		o.value('proxy', 'proxy');
		o.default = 'direct';
		o.rmempty = false;

		o = s.option(form.ListValue, 'real_dns_outbound', _('DNS outbound'));
		addProxyOutboundChoices(o);
		o.depends('real_dns_mode', 'proxy');
		o.rmempty = false;

		o = s.option(form.ListValue, 'real_dns_transport', _('DNS transport'));
		o.value('udp', _('UDP'));
		o.value('tcp', _('TCP'));
		o.value('tls', _('DNS over TLS'));
		o.value('https', _('DNS over HTTPS'));
		o.default = 'udp';
		o.rmempty = false;

		o = s.option(form.Value, 'real_dns_server', _('Server'));
		o.depends('dns_upstream_preset', 'custom');
		o.placeholder = '1.1.1.1:53';
		o.rmempty = false;

		o = s.option(form.Value, 'real_dns_server_name', _('DNS server name'));
		o.depends({ 'dns_upstream_preset': 'custom', 'real_dns_transport': 'tls' });
		o.placeholder = 'cloudflare-dns.com';

		o = s.option(form.Value, '_real_dns_doh', _('DoH server/path'));
		o.depends({ 'dns_upstream_preset': 'custom', 'real_dns_transport': 'https' });
		o.placeholder = 'cloudflare-dns.com/dns-query';
		o.cfgvalue = function() {
			var serverName = String(uci.get('neto', 'main', 'real_dns_server_name') || '').trim();
			var path = String(uci.get('neto', 'main', 'real_dns_path') || '').trim();

			if (serverName == '' && path == '')
				return '';
			if (path == '')
				path = '/dns-query';
			return serverName + path;
		};
		o.write = function(section_id, formvalue) {
			var parts = splitDoHValue(formvalue, uci.get('neto', 'main', 'real_dns_server_name'), uci.get('neto', 'main', 'real_dns_path'));

			uci.set('neto', section_id, 'real_dns_server_name', parts[0]);
			uci.set('neto', section_id, 'real_dns_path', parts[1]);
			uci.unset('neto', section_id, '_real_dns_doh');
		};

		o = s.option(form.ListValue, 'routing_mode', _('Routing mode'));
		o.value('custom', _('Custom'));
		o.value('global', _('Global'));
		o.default = 'custom';

		o = s.option(form.ListValue, 'default_outbound', _('Default outbound'));
		o.value('direct', 'direct');
		o.default = 'direct';

		return m.render();
	}
});
