'use strict';
'require baseclass';
'require uci';

function rulesTabVisible() {
	return String(uci.get('neto', 'main', 'routing_mode') || 'custom').trim() == 'custom';
}

function hideElement(el, hidden) {
	if (el)
		el.style.display = hidden ? 'none' : '';
}

function tabContainer(link) {
	var node = link;

	while (node && node.parentNode) {
		if (String(node.nodeName || '').toLowerCase() == 'li')
			return node;

		node = node.parentNode;
	}

	return link;
}

function updateRulesTab() {
	var links, hidden;

	if (typeof document == 'undefined')
		return;

	links = document.querySelectorAll('a[href]');
	hidden = !rulesTabVisible();

	for (var i = 0; i < links.length; i++) {
		var href = String(links[i].getAttribute('href') || '');

		if (href.indexOf('/admin/services/neto/rules') < 0 && href.indexOf('/neto/rules') < 0)
			continue;

		hideElement(tabContainer(links[i]), hidden);
	}
}

function syncRulesTab() {
	updateRulesTab();

	if (typeof window != 'undefined') {
		window.setTimeout(updateRulesTab, 0);
		window.setTimeout(updateRulesTab, 250);
	}
}

return baseclass.extend({
	syncRulesTab: syncRulesTab
});
