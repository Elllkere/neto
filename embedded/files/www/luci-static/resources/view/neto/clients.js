'use strict';
'require fs';
'require form';
'require uci';
'require view';
'require neto.i18n as netoI18n';

var _ = netoI18n.translate;

return view.extend({
	load: function() {
		return uci.load('neto');
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

		m = new form.Map('neto', _('neto'));

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
		o.value('proxy', _('Proxy'));
		o.value('direct', _('Direct'));
		o.default = 'default';

		return m.render();
	}
});
