'use strict';
'require fs';
'require ui';
'require view';
'require neto.i18n as netoI18n';

var _ = netoI18n.translate;

function netod(args) {
	return fs.exec('/usr/bin/netod', args).then(function(res) {
		if (res.code)
			throw new Error(res.stderr || res.stdout || _('Command failed'));

		return res.stdout || '';
	});
}

function displayLog(text) {
	text = String(text || '').replace(/\s+$/, '');
	return text || _('No logs yet');
}

return view.extend({
	load: function() {
		return netod([ 'logs', 'sing-box' ]).catch(function(err) {
			return err.message || String(err);
		});
	},

	handleRefresh: function() {
		return netod([ 'logs', 'sing-box' ])
			.then(L.bind(function(text) {
				this.logNode.textContent = displayLog(text);
			}, this))
			.catch(function(err) {
				ui.addNotification(null, E('p', {}, [ err.message || err ]), 'danger');
			});
	},

	handleClear: function(button) {
		button.disabled = true;
		return fs.exec('/usr/bin/netod', [ 'logs', 'sing-box', 'clear' ])
			.then(function(res) {
				if (res.code)
					throw new Error(res.stderr || res.stdout || _('Clear failed'));
			})
			.then(L.bind(function() {
				button.disabled = null;
				return this.handleRefresh();
			}, this))
			.catch(function(err) {
				button.disabled = null;
				ui.addNotification(null, E('p', {}, [ err.message || err ]), 'danger');
			});
	},

	render: function(logText) {
		var refreshButton, clearButton;

		refreshButton = E('button', {
			'class': 'cbi-button cbi-button-action',
			'click': L.bind(function(ev) {
				ev.preventDefault();
				return this.handleRefresh();
			}, this)
		}, _('Refresh'));

		clearButton = E('button', {
			'class': 'cbi-button cbi-button-remove',
			'click': L.bind(function(ev) {
				ev.preventDefault();
				return this.handleClear(clearButton);
			}, this)
		}, _('Clear'));

		this.logNode = E('pre', {
			'style': [
				'min-height:24rem',
				'max-height:70vh',
				'overflow:auto',
				'white-space:pre-wrap',
				'overflow-wrap:anywhere',
				'padding:12px',
				'border:1px solid var(--border-color-medium)',
				'border-radius:4px'
			].join(';')
		}, [ displayLog(logText) ]);

		return E([
			E('h2', {}, [ _('sing-box Logs') ]),
			E('div', { 'class': 'right' }, [
				refreshButton,
				' ',
				clearButton
			]),
			this.logNode
		]);
	}
});
