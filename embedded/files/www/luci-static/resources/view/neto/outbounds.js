'use strict';
'require fs';
'require form';
'require ui';
'require uci';
'require view';

function isReservedTag(tag) {
	return tag == 'direct' || tag == 'blocked' || tag == 'block' || tag == 'proxy_default';
}

function allowInsecureConfirm(ev, _section_id, value) {
	if (value == '1' && !confirm(_('Are you sure to allow insecure TLS?')))
		ev.target.firstElementChild.checked = null;
}

function dependsTLS(option) {
	option.depends({ 'type': 'vless', 'tls': '1' });
	option.depends({ 'type': 'trojan', 'tls': '1' });
	option.depends('type', 'hysteria2');
}

function dependsECH(option) {
	option.depends({ 'type': 'vless', 'tls': '1', 'ech': '1' });
	option.depends({ 'type': 'trojan', 'tls': '1', 'ech': '1' });
	option.depends({ 'type': 'hysteria2', 'ech': '1' });
}

function dependsReality(option) {
	option.depends({ 'type': 'vless', 'tls': '1', 'reality': '1' });
}

function dependsTransport(option, transport) {
	option.depends({ 'type': 'vless', 'transport': transport });
	option.depends({ 'type': 'trojan', 'transport': transport });
}

function normalizeOutbounds() {
	uci.sections('neto', 'outbound', function(section, sid) {
		var tag = String(uci.get('neto', sid, 'tag') || '').trim();
		var label = String(uci.get('neto', sid, 'label') || uci.get('neto', sid, 'name') || '').trim();

		if (sid == 'proxy_default' || tag == 'proxy_default')
			return;

		if (tag == '')
			uci.set('neto', sid, 'tag', sid);

		if (label == '')
			uci.set('neto', sid, 'label', sid);

		if (uci.get('neto', sid, 'type') == null)
			uci.set('neto', sid, 'type', 'vless');
	});
}

return view.extend({
	load: function() {
		return uci.load('neto');
	},

	handleSave: function() {
		return this.map.save(normalizeOutbounds).then(function() {
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

		s = m.section(form.GridSection, 'outbound', _('Outbounds'));
		s.anonymous = false;
		s.addremove = true;
		s.modaltitle = _('Outbound details');
		s.sectiontitle = function(section_id) {
			return uci.get('neto', section_id, 'label') || uci.get('neto', section_id, 'name') || section_id;
		};
		s.filter = function(section_id) {
			var tag = String(uci.get('neto', section_id, 'tag') || section_id || '').trim();
			return tag != 'proxy_default';
		};
		s.renderSectionAdd = function() {
			var el = form.GridSection.prototype.renderSectionAdd.apply(this, arguments);
			var nameEl = el.querySelector('.cbi-section-create-name');

			ui.addValidator(nameEl, 'uciname', true, L.bind(function(value) {
				var button = el.querySelector('.cbi-section-create > .cbi-button-add');
				var config = this.uciconfig || this.map.config;
				var tag = String(value || '').trim();

				if (tag == '') {
					button.disabled = true;
					return true;
				}

				if (isReservedTag(tag)) {
					button.disabled = true;
					return _('This tag is reserved');
				}

				if (uci.get(config, tag)) {
					button.disabled = true;
					return _('Expecting: %s').format(_('unique UCI identifier'));
				}

				button.disabled = null;
				return true;
			}, this), 'blur', 'keyup');

			return el;
		};

		o = s.option(form.Value, 'label', _('Name'));
		o.cfgvalue = function(section_id) {
			return uci.get('neto', section_id, 'label') || uci.get('neto', section_id, 'name') || section_id;
		};
		o.write = function(section_id, formvalue) {
			var label = String(formvalue || '').trim();
			uci.set('neto', section_id, 'label', label || section_id);
		};
		o.validate = function(section_id, value) {
			var label = String(value || section_id || '').trim();

			if (label == '')
				return _('Name is required');

			return true;
		};
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.ListValue, 'type', _('Type'));
		o.value('vless', _('VLESS'));
		o.value('hysteria2', _('Hysteria2'));
		o.value('shadowsocks', _('Shadowsocks'));
		o.value('trojan', _('Trojan'));
		o.default = 'vless';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.Value, 'server', _('Address'));
		o.datatype = 'host';
		o.depends('type', 'vless');
		o.depends('type', 'hysteria2');
		o.depends('type', 'shadowsocks');
		o.depends('type', 'trojan');
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.Value, 'port', _('Port'));
		o.datatype = 'port';
		o.depends('type', 'vless');
		o.depends('type', 'hysteria2');
		o.depends('type', 'shadowsocks');
		o.depends('type', 'trojan');
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.Value, 'uuid', _('UUID'));
		o.depends('type', 'vless');
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.ListValue, 'flow', _('Flow'));
		o.value('', _('None'));
		o.value('xtls-rprx-vision');
		o.depends('type', 'vless');
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.ListValue, 'hysteria_obfs_type', _('Obfuscate type'));
		o.value('', _('Disable'));
		o.value('salamander', _('Salamander'));
		o.depends('type', 'hysteria2');
		o.modalonly = true;

		o = s.option(form.Value, 'hysteria_obfs_password', _('Obfuscate password'));
		o.password = true;
		o.depends({ 'type': 'hysteria2', 'hysteria_obfs_type': /[\s\S]/ });
		o.modalonly = true;

		o = s.option(form.Value, 'hysteria_down_mbps', _('Max download speed'));
		o.datatype = 'uinteger';
		o.depends('type', 'hysteria2');
		o.modalonly = true;

		o = s.option(form.Value, 'hysteria_up_mbps', _('Max upload speed'));
		o.datatype = 'uinteger';
		o.depends('type', 'hysteria2');
		o.modalonly = true;

		o = s.option(form.Flag, 'tls', _('TLS'));
		o.enabled = '1';
		o.disabled = '0';
		o.depends('type', 'vless');
		o.depends('type', 'trojan');
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.Value, 'server_name', _('TLS SNI'));
		dependsTLS(o);
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'alpn', _('TLS ALPN'));
		dependsTLS(o);
		o.placeholder = 'h2';
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.Flag, 'insecure', _('Allow insecure'));
		o.enabled = '1';
		o.disabled = '0';
		dependsTLS(o);
		o.onchange = allowInsecureConfirm;
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.ListValue, 'tls_min_version', _('Minimum TLS version'));
		o.value('', _('Default'));
		o.value('1.0');
		o.value('1.1');
		o.value('1.2');
		o.value('1.3');
		dependsTLS(o);
		o.modalonly = true;

		o = s.option(form.ListValue, 'tls_max_version', _('Maximum TLS version'));
		o.value('', _('Default'));
		o.value('1.0');
		o.value('1.1');
		o.value('1.2');
		o.value('1.3');
		dependsTLS(o);
		o.modalonly = true;

		o = s.option(form.DynamicList, 'tls_cipher_suites', _('Cipher suites'));
		o.placeholder = 'TLS_AES_128_GCM_SHA256';
		dependsTLS(o);
		o.modalonly = true;

		o = s.option(form.Flag, 'ech', _('Enable ECH'));
		o.enabled = '1';
		o.disabled = '0';
		dependsTLS(o);
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DynamicList, 'ech_config', _('ECH config'));
		dependsECH(o);
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.Value, 'ech_config_path', _('ECH config path'));
		dependsECH(o);
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.ListValue, 'utls_fingerprint', _('uTLS fingerprint'));
		o.value('', _('Disable'));
		o.value('360');
		o.value('android');
		o.value('chrome');
		o.value('edge');
		o.value('firefox');
		o.value('ios');
		o.value('qq');
		o.value('random');
		o.value('randomized');
		o.value('safari');
		o.depends({ 'type': 'vless', 'tls': '1' });
		o.depends({ 'type': 'trojan', 'tls': '1' });
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.Flag, 'reality', _('REALITY'));
		o.enabled = '1';
		o.disabled = '0';
		o.depends({ 'type': 'vless', 'tls': '1' });
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.Value, 'reality_public_key', _('REALITY public key'));
		o.password = true;
		dependsReality(o);
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.Value, 'reality_short_id', _('REALITY short ID'));
		o.password = true;
		dependsReality(o);
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.ListValue, 'transport', _('Transport'));
		o.value('', _('None'));
		o.value('grpc', _('gRPC'));
		o.value('http', _('HTTP'));
		o.value('httpupgrade', _('HTTPUpgrade'));
		o.value('quic', _('QUIC'));
		o.value('ws', _('WebSocket'));
		o.depends('type', 'vless');
		o.depends('type', 'trojan');
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.Value, 'grpc_service_name', _('gRPC service name'));
		dependsTransport(o, 'grpc');
		o.modalonly = true;

		o = s.option(form.DynamicList, 'http_host', _('Host'));
		o.datatype = 'hostname';
		dependsTransport(o, 'http');
		o.modalonly = true;

		o = s.option(form.Value, 'httpupgrade_host', _('Host'));
		o.datatype = 'hostname';
		dependsTransport(o, 'httpupgrade');
		o.modalonly = true;

		o = s.option(form.Value, 'http_path', _('Path'));
		dependsTransport(o, 'http');
		dependsTransport(o, 'httpupgrade');
		o.modalonly = true;

		o = s.option(form.ListValue, 'http_method', _('Method'));
		o.value('', _('Default'));
		o.value('GET');
		o.value('PUT');
		dependsTransport(o, 'http');
		o.modalonly = true;

		o = s.option(form.Value, 'ws_host', _('Host'));
		dependsTransport(o, 'ws');
		o.modalonly = true;

		o = s.option(form.Value, 'ws_path', _('Path'));
		dependsTransport(o, 'ws');
		o.modalonly = true;

		o = s.option(form.Value, 'websocket_early_data', _('Early data'));
		o.datatype = 'uinteger';
		o.placeholder = '2048';
		dependsTransport(o, 'ws');
		o.modalonly = true;

		o = s.option(form.Value, 'websocket_early_data_header', _('Early data header name'));
		o.placeholder = 'Sec-WebSocket-Protocol';
		dependsTransport(o, 'ws');
		o.modalonly = true;

		o = s.option(form.ListValue, 'packet_encoding', _('Packet encoding'));
		o.value('', _('none'));
		o.value('packetaddr', _('packet addr'));
		o.value('xudp', _('XUDP'));
		o.depends('type', 'vless');
		o.modalonly = true;

		o = s.option(form.Value, 'password', _('Password'));
		o.password = true;
		o.depends('type', 'hysteria2');
		o.depends('type', 'shadowsocks');
		o.depends('type', 'trojan');
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.ListValue, 'method', _('Encrypt method'));
		o.value('2022-blake3-aes-128-gcm');
		o.value('2022-blake3-aes-256-gcm');
		o.value('2022-blake3-chacha20-poly1305');
		o.value('aes-128-gcm');
		o.value('aes-256-gcm');
		o.value('chacha20-ietf-poly1305');
		o.value('xchacha20-ietf-poly1305');
		o.value('aes-128-ctr');
		o.value('aes-192-ctr');
		o.value('aes-256-ctr');
		o.value('aes-128-cfb');
		o.value('aes-192-cfb');
		o.value('aes-256-cfb');
		o.value('chacha20');
		o.value('chacha20-ietf');
		o.value('rc4-md5');
		o.depends('type', 'shadowsocks');
		o.default = '2022-blake3-aes-128-gcm';
		o.rmempty = true;
		o.modalonly = true;

		return m.render();
	}
});
