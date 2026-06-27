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

function forceGeneralState() {
	uci.set('neto', 'main', 'fakeip_enabled', '1');
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
