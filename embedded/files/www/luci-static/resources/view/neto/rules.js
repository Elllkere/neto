'use strict';
'require fs';
'require form';
'require ui';
'require uci';
'require view';
'require neto.i18n as netoI18n';

var _ = netoI18n.translate;

function rewriteRuleState() {
	var n = 0;

	uci.sections('neto', 'rule', function(section, sid) {
		var outbound = uci.get('neto', sid, 'outbound');

		n++;
		uci.set('neto', sid, 'priority', String(n * 100));

		if (uci.get('neto', sid, 'enabled') == null)
			uci.set('neto', sid, 'enabled', '1');

		if (uci.get('neto', sid, 'action') == 'proxy')
			uci.set('neto', sid, 'dns_mode', 'fakeip');

		if (outbound == null || outbound == '' || outbound == 'proxy_default')
			uci.set('neto', sid, 'outbound', 'direct');
	});
}

function addOutboundChoices(option) {
	option.value('direct', _('Direct'));
	option.value('blocked', _('Blocked'));

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
		return this.map.save(rewriteRuleState).then(function() {
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

		m = new form.Map('neto', _('neto'));
		this.map = m;

		s = m.section(form.GridSection, 'rule', _('Rules'),
			_('These are literal string operations, not DNS-aware matching.') + ' ' +
			_('For root + subdomains, use Equals: example.com and Ends with: .example.com'));
		s.anonymous = true;
		s.addremove = true;
		s.sortable = true;
		s.modaltitle = _('Rule details');

		o = s.option(form.Flag, 'enabled', _('Enabled'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '1';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.Value, 'name', _('Name'));
		o.rmempty = false;

		o = s.option(form.ListValue, 'action', _('Action'));
		o.value('proxy', _('Proxy'));
		o.value('direct', _('Direct'));
		o.value('block', _('Block'));
		o.default = 'proxy';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.ListValue, 'outbound', _('Outbound'));
		addOutboundChoices(o);
		o.default = 'direct';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.DynamicList, 'domain_equals', _('Equals'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'domain_contains', _('Contains'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'domain_starts_with', _('Starts with'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'domain_ends_with', _('Ends with'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'exclude_domain_equals', _('Exclude equals'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'exclude_domain_contains', _('Exclude contains'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'exclude_domain_starts_with', _('Exclude starts with'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'exclude_domain_ends_with', _('Exclude ends with'));
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'file', _('IPv4 CIDR files'));
		o.rmempty = true;
		o.placeholder = '/etc/neto/providers/cloudflare-v4.txt';
		o.modalonly = true;

		return m.render();
	}
});
