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

		s = m.section(form.GridSection, 'provider', _('Providers'));
		s.anonymous = true;
		s.addremove = true;
		s.modaltitle = _('Provider details');

		o = s.option(form.Value, 'name', _('Name'));
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.DynamicList, 'file', _('IPv4 CIDR files'));
		o.placeholder = '/etc/neto/providers/example-v4.txt';
		o.rmempty = false;

		o = s.option(form.Value, 'description', _('Description'));
		o.rmempty = true;
		o.modalonly = true;

		return m.render();
	}
});
