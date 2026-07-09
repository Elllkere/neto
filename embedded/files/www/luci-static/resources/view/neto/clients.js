'use strict';
'require fs';
'require form';
'require ui';
'require uci';
'require view';
'require neto.i18n as netoI18n';
'require neto.ui as netoUI';

var _ = netoI18n.translate;

function addOutboundChoices(option) {
	option.value('', _('Auto'));

	uci.sections('neto', 'outbound', function(section, sid) {
		var tag = String(section.tag || sid || section['.name'] || '').trim();
		var label = String(section.label || section.name || tag).trim();

		if (tag == '' || tag == 'direct' || tag == 'blocked' || tag == 'block' || tag == 'proxy_default')
			return;

		option.value(tag, label || tag);
	});
}

function rewriteClientState() {
	uci.sections('neto', 'client', function(section, sid) {
		var policy = String(uci.get('neto', sid, 'policy') || 'default').trim();
		var outbound = String(uci.get('neto', sid, 'outbound') || '').trim();

		if (policy != 'proxy' || outbound == '' || outbound == 'direct' || outbound == 'blocked' || outbound == 'block' || outbound == 'proxy_default')
			uci.unset('neto', sid, 'outbound');
	});
}

return view.extend({
	load: function() {
		return uci.load('neto').then(function() {
			netoUI.syncRulesTab();
		});
	},

	handleSave: function() {
		return this.map.save(rewriteClientState).then(function() {
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

	render: function() {
		var m, s, o;

		netoUI.syncRulesTab();

		m = new form.Map('neto', _('neto'));
		this.map = m;

		s = m.section(form.GridSection, 'client', _('Clients'),
			_('Default follows general routing mode. Proxy forces non-reserved traffic through neto. Direct bypasses neto completely.'));
		s.anonymous = true;
		s.addremove = true;

		o = s.option(form.Value, 'name', _('Name'));
		o.rmempty = false;

		o = s.option(form.Value, 'ip', _('IPv4 address'));
		o.datatype = 'ip4addr';
		o.rmempty = false;

		o = s.option(form.ListValue, 'policy', _('Policy'));
		o.value('default', _('Default'));
		o.value('proxy', 'proxy');
		o.value('direct', 'direct');
		o.default = 'default';

		o = s.option(form.ListValue, 'outbound', _('Outbound'));
		addOutboundChoices(o);
		o.depends('policy', 'proxy');
		o.default = '';
		o.rmempty = true;
		o.editable = true;

		return m.render();
	}
});
