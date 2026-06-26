'use strict';
'require form';
'require view';

return view.extend({
	render: function() {
		var m, s, o;

		m = new form.Map('neto', _('neto'));

		s = m.section(form.GridSection, 'client', _('Clients'));
		s.anonymous = true;
		s.addremove = true;

		o = s.option(form.Value, 'name', _('Name'));
		o.rmempty = false;

		o = s.option(form.Value, 'ip', _('IPv4 address'));
		o.datatype = 'ip4addr';
		o.rmempty = false;

		o = s.option(form.ListValue, 'policy', _('Policy'),
			_('Default follows general routing mode. Proxy forces proxy_default. Direct bypasses neto completely.'));
		o.value('default', _('Default'));
		o.value('proxy', _('Proxy'));
		o.value('direct', _('Direct'));
		o.default = 'default';

		return m.render();
	}
});
