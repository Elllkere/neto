'use strict';
'require fs';
'require uci';
'require view';
'require neto.i18n as netoI18n';

var _ = netoI18n.translate;

return view.extend({
	load: function() {
		return uci.load('neto').then(function() {
			return fs.exec('/usr/bin/netod', [ 'debug' ]).catch(function(err) {
				return {
					code: -1,
					stdout: '',
					stderr: String(err)
				};
			});
		});
	},

	render: function(res) {
		var text = res.stdout || res.stderr || _('No status available');

		return E([
			E('h2', {}, [ _('Debug') ]),
			E('pre', {
				'style': 'white-space: pre-wrap; overflow-wrap: anywhere'
			}, [ text.trim() ])
		]);
	}
});
