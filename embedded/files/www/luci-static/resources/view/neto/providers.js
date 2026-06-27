'use strict';
'require fs';
'require form';
'require ui';
'require uci';
'require view';
'require neto.i18n as netoI18n';

var _ = netoI18n.translate;

function normalizeProviders() {
	uci.sections('neto', 'provider', function(section, sid) {
		if (uci.get('neto', sid, 'enabled') == null)
			uci.set('neto', sid, 'enabled', '1');

		if (uci.get('neto', sid, 'label') == null)
			uci.set('neto', sid, 'label', uci.get('neto', sid, 'name') || sid);

		if (uci.get('neto', sid, 'type') == null)
			uci.set('neto', sid, 'type', 'domain');

		if (uci.get('neto', sid, 'auto_update') == null)
			uci.set('neto', sid, 'auto_update', '0');

		if (uci.get('neto', sid, 'update_hour') == null)
			uci.set('neto', sid, 'update_hour', '0');

		if (uci.get('neto', sid, 'update_via') == null)
			uci.set('neto', sid, 'update_via', 'direct');
	});
}

function addProxyOutboundChoices(option) {
	option.value('', _('Auto'));

	uci.sections('neto', 'outbound', function(section, sid) {
		var tag = String(section.tag || sid || section['.name'] || '').trim();
		var label = String(section.label || section.name || tag).trim();

		if (tag == '' || tag == 'direct' || tag == 'blocked' || tag == 'block' || tag == 'proxy_default')
			return;

		option.value(tag, label || tag);
	});
}

return view.extend({
	load: function() {
		return uci.load('neto');
	},

	handleSave: function() {
		return this.map.save(normalizeProviders).then(function() {
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

	handleProviderUpdate: function(section_id) {
		return this.handleSave()
			.then(function() {
				return fs.exec('/sbin/uci', [ 'commit', 'neto' ]);
			})
			.then(function(res) {
				if (res.code)
					throw new Error(res.stderr || res.stdout || _('Commit failed'));

				return fs.exec('/usr/bin/netod', [ 'providers', 'update', section_id ]);
			})
			.then(function(res) {
				if (res.code)
					throw new Error(res.stderr || res.stdout || _('Update failed'));

				return fs.exec('/etc/init.d/neto', [ 'restart' ]);
			})
			.then(function() {
				window.location.reload();
			});
	},

	render: function() {
		var m, s, o;

		m = new form.Map('neto', _('neto'));
		this.map = m;

		s = m.section(form.GridSection, 'provider', _('Providers'));
		s.anonymous = false;
		s.addremove = true;
		s.modaltitle = _('Provider details');
		s.sectiontitle = function(section_id) {
			return uci.get('neto', section_id, 'label') || uci.get('neto', section_id, 'name') || section_id;
		};

		o = s.option(form.Flag, 'enabled', _('Enabled'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.Value, 'label', _('Name'));
		o.cfgvalue = function(section_id) {
			return uci.get('neto', section_id, 'label') || uci.get('neto', section_id, 'name') || section_id;
		};
		o.write = function(section_id, formvalue) {
			var label = String(formvalue || '').trim();
			uci.set('neto', section_id, 'label', label || section_id);
		};
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.ListValue, 'type', _('Type'));
		o.value('domain', _('Domains'));
		o.value('ip', _('IP/CIDR'));
		o.default = 'domain';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.Value, 'url', _('URL'));
		o.datatype = 'url';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.Flag, 'auto_update', _('Auto update'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '0';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.ListValue, 'update_hour', _('Update time'));
		for (var hour = 0; hour < 24; hour++)
			o.value(String(hour), _('%d:00').format(hour));
		o.default = '0';
		o.depends('auto_update', '1');
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.ListValue, 'update_via', _('Update via'));
		o.value('direct', _('Direct'));
		o.value('proxy', _('Proxy'));
		o.default = 'direct';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.ListValue, 'update_outbound', _('Update outbound'));
		addProxyOutboundChoices(o);
		o.depends('update_via', 'proxy');
		o.modalonly = true;

		o = s.option(form.DummyValue, 'item_count', _('Items'));
		o.cfgvalue = function(section_id) {
			return uci.get('neto', section_id, 'item_count') || '-';
		};

		o = s.option(form.DummyValue, 'last_update', _('Updated'));
		o.cfgvalue = function(section_id) {
			var value = uci.get('neto', section_id, 'last_update');
			var timestamp = Number(value);

			if (!timestamp)
				return '-';

			return new Date(timestamp * 1000).toLocaleString();
		};
		o.modalonly = true;

		o = s.option(form.DummyValue, 'local_path', _('Local cache'));
		o.cfgvalue = function(section_id) {
			return uci.get('neto', section_id, 'local_path') || '-';
		};
		o.modalonly = true;

		o = s.option(form.Value, 'description', _('Description'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.Button, '_update', _('Update'));
		o.inputstyle = 'action';
		o.onclick = L.bind(function(section_id) {
			return this.handleProviderUpdate(section_id);
		}, this);

		return m.render();
	}
});
