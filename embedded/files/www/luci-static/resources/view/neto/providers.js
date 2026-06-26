'use strict';
'require form';
'require view';

return view.extend({
	render: function() {
		var m, s, o;

		m = new form.Map('neto', _('neto'));

		s = m.section(form.GridSection, 'subnet_rule', _('Providers'));
		s.anonymous = true;
		s.addremove = true;

		o = s.option(form.Value, 'name', _('Name'));
		o.rmempty = false;

		o = s.option(form.DynamicList, 'file', _('IPv4 CIDR files'));
		o.placeholder = '/etc/neto/providers/example-v4.txt';
		o.rmempty = true;

		return m.render();
	}
});

