'use strict';
'require form';
'require view';

return view.extend({
	render: function() {
		var m, s, o;

		m = new form.Map('neto', _('neto'));

		s = m.section(form.GridSection, 'domain_rule', _('Domain rules'));
		s.anonymous = true;
		s.addremove = true;

		o = s.option(form.Value, 'name', _('Name'));
		o.rmempty = false;

		o = s.option(form.ListValue, 'mode', _('Mode'));
		o.value('fakeip', _('FakeIP'));
		o.default = 'fakeip';

		o = s.option(form.ListValue, 'outbound', _('Outbound'));
		o.value('proxy_default', _('Proxy default'));
		o.default = 'proxy_default';

		o = s.option(form.DynamicList, 'suffix', _('Suffix'));
		o.rmempty = true;

		s = m.section(form.GridSection, 'subnet_rule', _('Subnet rules'));
		s.anonymous = true;
		s.addremove = true;

		o = s.option(form.Value, 'name', _('Name'));
		o.rmempty = false;

		o = s.option(form.ListValue, 'outbound', _('Outbound'));
		o.value('proxy_default', _('Proxy default'));
		o.default = 'proxy_default';

		return m.render();
	}
});

