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
		var action = String(uci.get('neto', sid, 'action') || 'proxy').trim();
		var outbound = uci.get('neto', sid, 'outbound');
		var domainInput = String(uci.get('neto', sid, 'domain_input') || '').trim();
		var ipInput = String(uci.get('neto', sid, 'ip_input') || '').trim();

		n++;
		uci.set('neto', sid, 'priority', String(n * 100));

		if (uci.get('neto', sid, 'enabled') == null)
			uci.set('neto', sid, 'enabled', '1');

		uci.set('neto', sid, 'dns_mode', 'auto');

		if (action != 'proxy') {
			uci.set('neto', sid, 'outbound', 'direct');
		} else if (outbound == null || outbound == '' || outbound == 'direct' || outbound == 'blocked' || outbound == 'block' || outbound == 'proxy_default') {
			uci.unset('neto', sid, 'outbound');
		}

		if (domainInput == '') {
			if (optionValues(sid, 'domain_provider').length > 0)
				domainInput = 'provider';
			else if (optionValues(sid, 'domain_file').length > 0)
				domainInput = 'file';
		}
		if (domainInput != 'text' && domainInput != 'provider' && domainInput != 'file')
			domainInput = 'fields';
		uci.set('neto', sid, 'domain_input', domainInput);

		if (domainInput == 'provider') {
			unsetRuleOptions(sid, [
				'domain_equals', 'domain_contains', 'domain_starts_with', 'domain_ends_with',
				'exclude_domain_equals', 'exclude_domain_contains', 'exclude_domain_starts_with', 'exclude_domain_ends_with',
				'domain_file'
			]);
		} else if (domainInput == 'file') {
			unsetRuleOptions(sid, [
				'domain_equals', 'domain_contains', 'domain_starts_with', 'domain_ends_with',
				'exclude_domain_equals', 'exclude_domain_contains', 'exclude_domain_starts_with', 'exclude_domain_ends_with',
				'domain_provider'
			]);
		} else {
			uci.unset('neto', sid, 'domain_provider');
			uci.unset('neto', sid, 'domain_file');
		}

		if (ipInput == '') {
			if (optionValues(sid, 'ip_provider').length > 0)
				ipInput = 'provider';
			else if (optionValues(sid, 'ip_file').length > 0 || optionValues(sid, 'file').length > 0)
				ipInput = 'file';
			else
				ipInput = 'list';
		}
		if (ipInput != 'text' && ipInput != 'provider' && ipInput != 'file')
			ipInput = 'list';
		uci.set('neto', sid, 'ip_input', ipInput);

		if (ipInput == 'provider') {
			unsetRuleOptions(sid, [ 'ip_cidr', 'ip_file', 'file' ]);
		} else if (ipInput == 'file') {
			unsetRuleOptions(sid, [ 'ip_cidr', 'ip_provider', 'file' ]);
		} else {
			uci.unset('neto', sid, 'ip_provider');
			uci.unset('neto', sid, 'ip_file');
			uci.unset('neto', sid, 'file');
		}
	});
}

function addOutboundChoices(option) {
	option.value('', _('Select outbound'));

	uci.sections('neto', 'outbound', function(section, sid) {
		var tag = String(section.tag || sid || section['.name'] || '').trim();
		var label = String(section.label || section.name || tag).trim();

		if (tag == '' || tag == 'direct' || tag == 'blocked' || tag == 'block' || tag == 'proxy_default')
			return;

		option.value(tag, label || tag);
	});
}

function toArray(value) {
	if (Array.isArray(value))
		return value;
	if (value == null || value == '')
		return [];
	return [ value ];
}

function cleanValues(values) {
	var seen = {};
	var out = [];

	values = toArray(values);
	for (var i = 0; i < values.length; i++) {
		var value = String(values[i] || '').trim();

		if (value == '' || seen[value])
			continue;

		seen[value] = true;
		out.push(value);
	}

	return out;
}

function splitTextValues(value) {
	var values = [];
	var lines = String(value || '').replace(/\r/g, '\n').split('\n');

	for (var i = 0; i < lines.length; i++) {
		var line = lines[i];
		var comment = line.indexOf('#');

		if (comment >= 0)
			line = line.slice(0, comment);

		values.push(line);
	}

	return cleanValues(values);
}

function optionValues(section_id, option) {
	return cleanValues(uci.get('neto', section_id, option));
}

function setListOption(section_id, option, values) {
	values = cleanValues(values);
	if (values.length > 0)
		uci.set('neto', section_id, option, values);
	else
		uci.unset('neto', section_id, option);
}

function unsetRuleOptions(section_id, options) {
	for (var i = 0; i < options.length; i++)
		uci.unset('neto', section_id, options[i]);
}

function addDynamicList(section, option, title, modeOption, modeValue, placeholder) {
	var o = section.option(form.DynamicList, option, title);

	o.rmempty = true;
	o.modalonly = true;
	if (placeholder)
		o.placeholder = placeholder;
	o.depends(modeOption, modeValue);

	return o;
}

function addTextList(section, option, target, title, modeOption, modeValue, placeholder) {
	var o = section.option(form.TextValue, option, title);

	o.rows = 6;
	o.rmempty = true;
	o.modalonly = true;
	o.depends(modeOption, modeValue);
	if (placeholder)
		o.placeholder = placeholder;
	o.cfgvalue = function(section_id) {
		return optionValues(section_id, target).join('\n');
	};
	o.write = function(section_id, formvalue) {
		setListOption(section_id, target, splitTextValues(formvalue));
	};

	return o;
}

function addProviderChoices(option, type) {
	uci.sections('neto', 'provider', function(section, sid) {
		var name = String(sid || section['.name'] || '').trim();
		var label = String(section.label || section.name || name).trim();
		var providerType = String(section.type || '').trim();

		if (name == '' || providerType != type)
			return;

		option.value(name, label || name);
	});
}

function addProviderList(section, option, title, providerType, modeOption, modeValue) {
	var o = section.option(form.DynamicList, option, title);

	o.rmempty = true;
	o.modalonly = true;
	o.depends(modeOption, modeValue);
	addProviderChoices(o, providerType);

	return o;
}

function packetProtoValue(section_id) {
	var values = optionValues(section_id, 'proto');
	var hasTCP = values.indexOf('tcp') >= 0;
	var hasUDP = values.indexOf('udp') >= 0;

	if (hasTCP && hasUDP)
		return 'tcp_udp';
	if (hasTCP)
		return 'tcp';
	if (hasUDP)
		return 'udp';
	return 'any';
}

function writePacketProto(section_id, formvalue) {
	switch (formvalue) {
	case 'tcp':
		setListOption(section_id, 'proto', [ 'tcp' ]);
		break;
	case 'udp':
		setListOption(section_id, 'proto', [ 'udp' ]);
		break;
	case 'tcp_udp':
		setListOption(section_id, 'proto', [ 'tcp', 'udp' ]);
		break;
	default:
		uci.unset('neto', section_id, 'proto');
	}
}

function validatePortMatch(section_id, value) {
	var values = Array.isArray(value) ? value : [ value ];

	for (var i = 0; i < values.length; i++) {
		var port = String(values[i] || '').trim();
		var parts, start, end;

		if (port == '')
			continue;

		if (!/^[0-9]+(-[0-9]+)?$/.test(port))
			return _('Port must be a number or range, for example 443 or 1000-2000');

		parts = port.split('-');
		start = parseInt(parts[0], 10);
		end = parts.length > 1 ? parseInt(parts[1], 10) : start;

		if (start < 1 || start > 65535 || end < 1 || end > 65535)
			return _('Port must be between 1 and 65535');

		if (start > end)
			return _('Port range start must be less than or equal to range end');
	}

	return true;
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
		o.value('proxy', 'proxy');
		o.value('direct', 'direct');
		o.value('block', 'block');
		o.default = 'proxy';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.ListValue, 'outbound', _('Outbound'));
		addOutboundChoices(o);
		o.depends('action', 'proxy');
		o.default = '';
		o.rmempty = true;
		o.editable = true;

		o = s.option(form.ListValue, 'domain_input', _('Domain input'));
		o.value('fields', _('Fields'));
		o.value('text', _('Textbox'));
		o.value('file', _('File paths'));
		o.value('provider', _('Providers'));
		o.default = 'fields';
		o.rmempty = false;
		o.modalonly = true;
		o.cfgvalue = function(section_id) {
			var value = String(uci.get('neto', section_id, 'domain_input') || '').trim();

			if (value != '')
				return value;
			if (optionValues(section_id, 'domain_provider').length > 0)
				return 'provider';
			if (optionValues(section_id, 'domain_file').length > 0)
				return 'file';
			return 'fields';
		};

		addDynamicList(s, 'domain_equals', _('Equals'), 'domain_input', 'fields');
		addDynamicList(s, 'domain_contains', _('Contains'), 'domain_input', 'fields');
		addDynamicList(s, 'domain_starts_with', _('Starts with'), 'domain_input', 'fields');
		addDynamicList(s, 'domain_ends_with', _('Ends with'), 'domain_input', 'fields');
		addDynamicList(s, 'exclude_domain_equals', _('Exclude equals'), 'domain_input', 'fields');
		addDynamicList(s, 'exclude_domain_contains', _('Exclude contains'), 'domain_input', 'fields');
		addDynamicList(s, 'exclude_domain_starts_with', _('Exclude starts with'), 'domain_input', 'fields');
		addDynamicList(s, 'exclude_domain_ends_with', _('Exclude ends with'), 'domain_input', 'fields');

		addTextList(s, '_domain_equals_text', 'domain_equals', _('Equals text'), 'domain_input', 'text', 'example.com\nexample.org');
		addTextList(s, '_domain_contains_text', 'domain_contains', _('Contains text'), 'domain_input', 'text', 'youtube');
		addTextList(s, '_domain_starts_with_text', 'domain_starts_with', _('Starts with text'), 'domain_input', 'text', 'www.');
		addTextList(s, '_domain_ends_with_text', 'domain_ends_with', _('Ends with text'), 'domain_input', 'text', '.example.com');
		addTextList(s, '_exclude_domain_equals_text', 'exclude_domain_equals', _('Exclude equals text'), 'domain_input', 'text');
		addTextList(s, '_exclude_domain_contains_text', 'exclude_domain_contains', _('Exclude contains text'), 'domain_input', 'text');
		addTextList(s, '_exclude_domain_starts_with_text', 'exclude_domain_starts_with', _('Exclude starts with text'), 'domain_input', 'text');
		addTextList(s, '_exclude_domain_ends_with_text', 'exclude_domain_ends_with', _('Exclude ends with text'), 'domain_input', 'text');

		addProviderList(s, 'domain_provider', _('Domain providers'), 'domain', 'domain_input', 'provider');
		addDynamicList(s, 'domain_file', _('Domain file paths'), 'domain_input', 'file', '/etc/neto/domains.txt');

		o = s.option(form.ListValue, 'ip_input', _('IP input'));
		o.value('list', _('List'));
		o.value('text', _('Textbox'));
		o.value('file', _('File paths'));
		o.value('provider', _('Providers'));
		o.default = 'list';
		o.rmempty = false;
		o.modalonly = true;
		o.cfgvalue = function(section_id) {
			var value = String(uci.get('neto', section_id, 'ip_input') || '').trim();

			if (value != '')
				return value;
			if (optionValues(section_id, 'ip_provider').length > 0)
				return 'provider';
			if (optionValues(section_id, 'ip_file').length > 0 || optionValues(section_id, 'file').length > 0)
				return 'file';
			return 'list';
		};

		addDynamicList(s, 'ip_cidr', _('IP/CIDR list'), 'ip_input', 'list', '1.1.1.1');
		addTextList(s, '_ip_cidr_text', 'ip_cidr', _('IP/CIDR text'), 'ip_input', 'text', '1.1.1.1\n8.8.8.0/24');
		addProviderList(s, 'ip_provider', _('IP providers'), 'ip', 'ip_input', 'provider');
		addDynamicList(s, 'ip_file', _('IP/CIDR file paths'), 'ip_input', 'file', '/etc/neto/ipv4-cidr.txt');

		o = s.option(form.DummyValue, '_packet_match', _('Advanced packet match'),
			_('Port matching is packet-level. It applies only to provider/CIDR/IP matches, not to DNS/FakeIP domain matching.'));
		o.modalonly = true;
		o.cfgvalue = function() {
			return '';
		};

		o = s.option(form.ListValue, '_packet_proto', _('Protocol'));
		o.value('any', _('Any'));
		o.value('tcp', _('TCP'));
		o.value('udp', _('UDP'));
		o.value('tcp_udp', _('TCP+UDP'));
		o.default = 'any';
		o.rmempty = false;
		o.modalonly = true;
		o.cfgvalue = packetProtoValue;
		o.write = writePacketProto;

		o = s.option(form.DynamicList, 'src_port', _('Source ports'),
			_('Source ports are client-side ports chosen by the LAN device. Usually leave empty. Syntax: 443 or 1000-2000.'));
		o.placeholder = '1000-2000';
		o.rmempty = true;
		o.modalonly = true;
		o.validate = validatePortMatch;

		o = s.option(form.DynamicList, 'dst_port', _('Destination ports'),
			_('Destination ports are service ports on the remote IP, for example 443 for HTTPS or 53 for DNS. Syntax: 443 or 1000-2000.'));
		o.placeholder = '443';
		o.rmempty = true;
		o.modalonly = true;
		o.validate = validatePortMatch;

		return m.render();
	}
});
