'use strict';
'require fs';
'require view';

return view.extend({
	load: function() {
		return fs.exec('/usr/bin/netod', [ 'status' ]).catch(function(err) {
			return {
				code: -1,
				stdout: '',
				stderr: String(err)
			};
		});
	},

	render: function(res) {
		var text = res.stdout || res.stderr || _('No status available');

		return E([
			E('h2', {}, [ _('neto') ]),
			E('pre', {
				'style': 'white-space: pre-wrap; overflow-wrap: anywhere'
			}, [ text.trim() ])
		]);
	}
});

