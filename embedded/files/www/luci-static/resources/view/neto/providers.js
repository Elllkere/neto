'use strict';
'require fs';
'require form';
'require ui';
'require uci';
'require view';
'require neto.i18n as netoI18n';

var _ = netoI18n.translate;
var communityDomainProviders = [
	{
		section: 'community_telegram_domains',
		label: 'Telegram domains',
		url: 'https://raw.githubusercontent.com/itdoginfo/allow-domains/refs/heads/main/Services/telegram.lst'
	},
	{
		section: 'community_tiktok_domains',
		label: 'TikTok domains',
		url: 'https://raw.githubusercontent.com/itdoginfo/allow-domains/refs/heads/main/Services/tiktok.lst'
	},
	{
		section: 'community_twitter_domains',
		label: 'Twitter domains',
		url: 'https://raw.githubusercontent.com/itdoginfo/allow-domains/refs/heads/main/Services/twitter.lst'
	},
	{
		section: 'community_youtube_domains',
		label: 'YouTube domains',
		url: 'https://raw.githubusercontent.com/itdoginfo/allow-domains/refs/heads/main/Services/youtube.lst'
	},
	{
		section: 'community_meta_domains',
		label: 'Meta domains',
		url: 'https://raw.githubusercontent.com/itdoginfo/allow-domains/refs/heads/main/Services/meta.lst'
	},
	{
		section: 'community_discord_domains',
		label: 'Discord domains',
		url: 'https://raw.githubusercontent.com/itdoginfo/allow-domains/refs/heads/main/Services/discord.lst'
	},
	{
		section: 'community_anime_domains',
		label: 'Anime domains',
		url: 'https://raw.githubusercontent.com/itdoginfo/allow-domains/refs/heads/main/Categories/anime.lst'
	}
];

function normalizeProviders() {
	uci.sections('neto', 'provider', function(section, sid) {
		if (uci.get('neto', sid, 'enabled') == null)
			uci.set('neto', sid, 'enabled', '1');

		if (uci.get('neto', sid, 'label') == null)
			uci.set('neto', sid, 'label', uci.get('neto', sid, 'name') || sid);

		if (uci.get('neto', sid, 'type') == null)
			uci.set('neto', sid, 'type', 'domain');

		if (uci.get('neto', sid, 'source') == null)
			uci.set('neto', sid, 'source', 'url');

		if (uci.get('neto', sid, 'auto_update') == null)
			uci.set('neto', sid, 'auto_update', '0');

		if (uci.get('neto', sid, 'update_schedule') == null)
			uci.set('neto', sid, 'update_schedule', 'time');

		if (uci.get('neto', sid, 'update_hour') == null)
			uci.set('neto', sid, 'update_hour', '0');

		if (uci.get('neto', sid, 'update_minute') == null)
			uci.set('neto', sid, 'update_minute', '5');

		if (uci.get('neto', sid, 'update_interval_minutes') == null)
			uci.set('neto', sid, 'update_interval_minutes', '360');

		if (uci.get('neto', sid, 'update_via') == null)
			uci.set('neto', sid, 'update_via', 'direct');
	});
}

function cleanValues(value) {
	var values = [];

	if (Array.isArray(value)) {
		values = value;
	} else if (value != null) {
		values = String(value).split(/\s+/);
	}

	var out = [];
	var seen = {};
	for (var i = 0; i < values.length; i++) {
		var item = String(values[i] || '').trim();

		if (item == '' || seen[item])
			continue;

		seen[item] = true;
		out.push(item);
	}

	return out;
}

function optionValues(section_id, option) {
	return cleanValues(uci.get('neto', section_id, option));
}

function activeProviderNames() {
	var names = {};

	uci.sections('neto', 'provider', function(section, sid) {
		if (uci.get('neto', sid, 'enabled') == '0')
			return;

		names[sid] = true;

		var name = String(uci.get('neto', sid, 'name') || '').trim();
		if (name != '')
			names[name] = true;
	});

	return names;
}

function referencedProviders(section_id) {
	return cleanValues([]
		.concat(optionValues(section_id, 'domain_provider'))
		.concat(optionValues(section_id, 'ip_provider'))
		.concat(optionValues(section_id, 'provider')));
}

function validateProviderReferences() {
	var providers = activeProviderNames();
	var error = null;

	uci.sections('neto', 'rule', function(section, sid) {
		if (error != null || uci.get('neto', sid, 'enabled') == '0')
			return;

		var ruleName = String(uci.get('neto', sid, 'name') || sid).trim();
		var refs = referencedProviders(sid);

		for (var i = 0; i < refs.length; i++) {
			if (!providers[refs[i]]) {
				error = _('Rule "%s" references missing or disabled provider "%s". Remove the provider from the rule before deleting or disabling it.').format(ruleName, refs[i]);
				return;
			}
		}
	});

	if (error != null)
		throw new Error(error);
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

function addUpdateIntervalChoices(option) {
	option.value('15', _('Every 15 minutes'));
	option.value('30', _('Every 30 minutes'));
	option.value('60', _('Every hour'));
	option.value('120', _('Every 2 hours'));
	option.value('180', _('Every 3 hours'));
	option.value('360', _('Every 6 hours'));
	option.value('720', _('Every 12 hours'));
	option.value('1440', _('Every day'));
}

function providerURLExists(url) {
	var exists = false;

	url = String(url || '').trim();
	uci.sections('neto', 'provider', function(section, sid) {
		if (String(uci.get('neto', sid, 'url') || '').trim() == url)
			exists = true;
	});

	return exists;
}

function uniqueProviderSection(base) {
	var section = base;
	var n = 2;

	while (uci.get('neto', section) != null) {
		section = base + '_' + n;
		n++;
	}

	return section;
}

function addCommunityDomainProvider(def) {
	var section;

	if (providerURLExists(def.url))
		return false;

	section = uniqueProviderSection(def.section);
	uci.set('neto', section, 'provider');
	uci.set('neto', section, 'enabled', '1');
	uci.set('neto', section, 'label', def.label);
	uci.set('neto', section, 'type', 'domain');
	uci.set('neto', section, 'source', 'url');
	uci.set('neto', section, 'url', def.url);
	uci.set('neto', section, 'auto_update', '0');
	uci.set('neto', section, 'update_schedule', 'time');
	uci.set('neto', section, 'update_hour', '0');
	uci.set('neto', section, 'update_minute', '5');
	uci.set('neto', section, 'update_interval_minutes', '360');
	uci.set('neto', section, 'update_via', 'direct');
	return true;
}

return view.extend({
	load: function() {
		return uci.load('neto');
	},

	handleSave: function() {
		return this.map.save(normalizeProviders)
			.then(function() {
				validateProviderReferences();
				return ui.changes.init();
			})
			.catch(function(err) {
				ui.addNotification(null, E('p', {}, [ err.message || err ]), 'danger');
				if (err != null && typeof err == 'object')
					err.notified = true;
				return Promise.reject(err);
			});
	},

	handleSaveCommitConfig: function() {
		return this.handleSave()
			.then(function() {
				return fs.exec('/sbin/uci', [ 'commit', 'neto' ]);
			})
			.then(function(res) {
				if (res.code)
					throw new Error(res.stderr || res.stdout || _('Commit failed'));

				return uci.load('neto');
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
		return this.handleSaveCommitConfig()
			.then(function() {
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

	handleAddCommunityProviders: function() {
		var added = 0;

		for (var i = 0; i < communityDomainProviders.length; i++) {
			if (addCommunityDomainProvider(communityDomainProviders[i]))
				added++;
		}

		if (added == 0) {
			ui.addNotification(null, E('p', {}, [ _('Community lists already exist') ]), 'info');
			return Promise.resolve();
		}

		return this.handleSaveCommitConfig()
			.then(function() {
				return fs.exec('/etc/init.d/neto', [ 'restart' ]);
			})
			.then(function() {
				window.location.reload();
			});
	},

	render: function() {
		var m, s, o, self;

		m = new form.Map('neto', _('neto'));
		this.map = m;
		self = this;

		s = m.section(form.GridSection, 'provider', _('Providers'));
		s.anonymous = false;
		s.addremove = true;
		s.modaltitle = _('Provider details');
		s.sectiontitle = function(section_id) {
			return uci.get('neto', section_id, 'label') || uci.get('neto', section_id, 'name') || section_id;
		};
		s.renderSectionAdd = function() {
			var el = form.GridSection.prototype.renderSectionAdd.apply(this, arguments);

			el.appendChild(E('button', {
				'class': 'cbi-button cbi-button-action',
				'click': function(ev) {
					ev.preventDefault();
					return self.handleAddCommunityProviders().catch(function(err) {
						ui.addNotification(null, E('p', {}, [ err.message || err ]), 'danger');
					});
				}
			}, _('Add community lists')));

			return el;
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

		o = s.option(form.ListValue, 'source', _('Source'));
		o.value('url', _('URL'));
		o.value('script', _('Script'));
		o.default = 'url';
		o.rmempty = false;

		o = s.option(form.Value, 'url', _('URL'));
		o.datatype = 'url';
		o.depends('source', 'url');
		o.rmempty = true;

		o = s.option(form.DummyValue, '_url_help', _('URL provider notes'));
		o.cfgvalue = function() {
			return _('Use a plain text URL with one domain, IPv4 address, or IPv4 CIDR per line. Use Script for JSON feeds or custom filtering.');
		};
		o.depends('source', 'url');
		o.modalonly = true;

		o = s.option(form.Value, 'script_path', _('Script path'));
		o.placeholder = '/usr/share/neto/providers/custom.sh';
		o.depends('source', 'script');
		o.rmempty = true;
		o.modalonly = true;

		o = s.option(form.DummyValue, '_script_help', _('Script provider notes'));
		o.cfgvalue = function() {
			return _('Use an absolute executable path. The script must print the final list to stdout or write it to NETO_PROVIDER_OUTPUT; one item per line. In proxy mode, neto exports NETO_PROVIDER_PROXY and HTTP_PROXY/HTTPS_PROXY/ALL_PROXY to the script.');
		};
		o.depends('source', 'script');
		o.modalonly = true;

		o = s.option(form.Flag, 'auto_update', _('Auto update'));
		o.enabled = '1';
		o.disabled = '0';
		o.default = '0';
		o.rmempty = false;
		o.editable = true;

		o = s.option(form.ListValue, 'update_schedule', _('Schedule'));
		o.value('time', _('Fixed time'));
		o.value('interval', _('Interval'));
		o.default = 'time';
		o.depends('auto_update', '1');
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.ListValue, 'update_hour', _('Update time'));
		for (var hour = 0; hour < 24; hour++)
			o.value(String(hour), _('%d:00').format(hour));
		o.default = '0';
		o.depends({ 'auto_update': '1', 'update_schedule': 'time' });
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.ListValue, 'update_minute', _('Update minute'));
		for (var minute = 0; minute < 60; minute++)
			o.value(String(minute), String(minute));
		o.default = '5';
		o.depends({ 'auto_update': '1', 'update_schedule': 'time' });
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.ListValue, 'update_interval_minutes', _('Update interval'));
		addUpdateIntervalChoices(o);
		o.default = '360';
		o.depends({ 'auto_update': '1', 'update_schedule': 'interval' });
		o.rmempty = false;
		o.modalonly = true;

		o = s.option(form.ListValue, 'update_via', _('Update via'));
		o.value('direct', 'direct');
		o.value('proxy', 'proxy');
		o.default = 'direct';
		o.rmempty = false;

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

		o = s.option(form.DummyValue, 'local_path', _('Local cache'));
		o.cfgvalue = function(section_id) {
			return uci.get('neto', section_id, 'local_path') || '-';
		};
		o.modalonly = true;

		o = s.option(form.Button, '_update', _('Update'));
		o.inputstyle = 'action';
		o.inputtitle = _('Update');
		o.cfgvalue = function() {
			return true;
		};
		o.modalonly = true;
		o.onclick = L.bind(function(ev, section_id) {
			return this.handleProviderUpdate(section_id).catch(function(err) {
				if (!(err != null && typeof err == 'object' && err.notified))
					ui.addNotification(null, E('p', {}, [ err.message || err ]), 'danger');
			});
		}, this);

		return m.render();
	}
});
