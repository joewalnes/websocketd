// Copyright 2013 Joe Walnes and the websocketd team.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package libwebsocketd

// Although this isn't particularly elegant, it's the simplest
// way to embed the console content into the binary.

// Note that the console is served by a single HTML file containing
// all CSS and JS inline.
// We can get by without jQuery or Bootstrap for this one ;).

const (
	defaultConsoleContent = `

<!--
websocketd console

Full documentation at http://websocketd.com/

{{license}}
-->

<!DOCTYPE html>
<meta charset="utf8">
<title>websocketd console</title>

<style>
	.template {
		display: none !important;
	}
	body, input {
		font-family: dejavu sans mono, Menlo, Monaco, Consolas, Lucida Console, tahoma, arial;
		font-size: 13px;
	}
	body {
		margin: 0;
	}
	.header {
		background-color: #efefef;
		padding: 2px;
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		height: 32px;
	}
	.header button {
		font-size: 19px;
		width: 30px;
		margin: 2px 2px 0 2px;
		padding: 0;
		float: left;
	}
	.header .url-holder {
		position: absolute;
		left: 38px;
		top: 4px;
		right: 14px;
		bottom: 9px;
	}
	.header .url {
		border: 1px solid #999;
		background-color: #fff;
		width: 100%;
		height: 100%;
		border-radius: 2px;
		padding-left: 4px;
		padding-right: 4px;
	}
	.messages {
		overflow-y: scroll;
		position: absolute;
		left: 0;
		right: 0;
		top: 36px;
		bottom: 0;
		border-top: 1px solid #ccc;
	}
	.message {
		border-bottom: 1px solid #bbb;
		padding: 2px;
	}
	.message-type {
		font-weight: bold;
		position: absolute;
		width: 80px;
		display: block;
	}
	.message-data {
		margin-left: 90px;
		display: block;
		word-wrap: break-word;
		white-space: pre;
	}
	.type-input,
	.type-send {
		background-color: #ffe;
	}
	.type-onmessage {
		background-color: #eef;
	}
	.type-open,
	.type-onopen {
		background-color: #efe;
	}
	.type-close,
	.type-onclose {
		background-color: #fee;
	}
	.type-onerror,
	.type-exception {
		background-color: #333;
		color: #f99;
	}
	.type-send .message-type,
	.type-onmessage .message-type {
		opacity: 0.2;
	}
	.type-input .message-type {
		color: #090;
	}
	.send-input {
		width: 100%;
		border: 0;
		padding: 0;
		margin: -1px;
		background-color: inherit;
	}
	.send-input:focus {
		outline: none;
	}
</style>

<header class="header">
	<button class="disconnect" title="Disconnect" style="display:none">&times;</button>
	<button class="connect" title="Connect" style="display:none">&#x2714;</button>
	<div class="url-holder">
		<input class="url" type="text" value="{{addr}}" spellcheck="false">
	</div>
</header>

<section class="messages">
	<div class="message template">
		<span class="message-type"></span>
		<span class="message-data"></span>
	</div>
	<div class="message type-input">
		<span class="message-type">send &#xbb;</span>
		<span class="message-data"><input type="text" class="send-input" spellcheck="false"></span>
	</div>
</section>

<script>

	var ws = null;

	function ready() {
		select('.connect').style.display = 'block';
		select('.disconnect').style.display = 'none';

		select('.connect').addEventListener('click', function() {
			connect(select('.url').value);
		});
		select('.disconnect').addEventListener('click', function() {
			disconnect();
		});

		select('.url').focus();
		select('.url').addEventListener('keydown', function(ev) {
			var code = ev.which || ev.keyCode;
			// Enter key pressed
			if (code  == 13) { 			
				updatePageUrl();
				connect(select('.url').value);
			}
		});
		select('.url').addEventListener('change', updatePageUrl);

		select('.send-input').addEventListener('keydown', function(ev) {
			var code = ev.which || ev.keyCode;
			// Enter key pressed
			if (code == 13) { 
				var msg = select('.send-input').value;
				select('.send-input').value = '';
				send(msg);
			}
			// Up key pressed
			if (code == 38) {
				moveThroughSendHistory(1);
			}
			// Down key pressed
			if (code == 40) {
				moveThroughSendHistory(-1);
			}
		});
		window.addEventListener('popstate', updateWebSocketUrl);
		updateWebSocketUrl();
	}

	function updatePageUrl() {
		var match = select('.url').value.match(new RegExp('^(ws)(s)?://([^/]*)(/.*)$'));
		if (match) {
			var pageUrlSuffix = match[4];
			if (history.state != pageUrlSuffix) {
				history.pushState(pageUrlSuffix, pageUrlSuffix, pageUrlSuffix);
			}
		}
	}

	function updateWebSocketUrl() {
		var match = location.href.match(new RegExp('^(http)(s)?://([^/]*)(/.*)$'));
		if (match) {
			var wsUrl = 'ws' + (match[2] || '') + '://' + match[3] + match[4];
			select('.url').value = wsUrl;
		}
	}

	function appendMessage(type, data) {
		var template = select('.message.template');
		var el = template.parentElement.insertBefore(template.cloneNode(true), select('.message.type-input'));
		el.classList.remove('template');
		el.classList.add('type-' + type.toLowerCase());
		el.querySelector('.message-type').textContent = type;
		el.querySelector('.message-data').textContent = data || '';
		el.querySelector('.message-data').innerHTML += '&nbsp;';
		el.scrollIntoView(true);
	}

	function connect(url) {
		function action() {
			appendMessage('open', url);
			try {
				ws = new WebSocket(url);
			} catch (ex) {
				appendMessage('exception', 'Cannot connect: ' + ex);
				return;
			}

			select('.connect').style.display = 'none';
			select('.disconnect').style.display = 'block';

			ws.addEventListener('open', function(ev) {
				appendMessage('onopen');
			});
			ws.addEventListener('close', function(ev) {
				select('.connect').style.display = 'block';
				select('.disconnect').style.display = 'none';
				appendMessage('onclose', '[Clean: ' + ev.wasClean + ', Code: ' + ev.code + ', Reason: ' + (ev.reason || 'none') + ']');
				ws = null;
				select('.url').focus();
			});
			ws.addEventListener('message', function(ev) {
				appendMessage('onmessage', ev.data);
			});
			ws.addEventListener('error', function(ev) {
				appendMessage('onerror');
			});

			select('.send-input').focus();
		}

		if (ws) {
			ws.addEventListener('close', function(ev) {
				action();
			});
			disconnect();
		} else {
			action();
		}
	}

	function disconnect() {
		if (ws) {
			appendMessage('close');
			ws.close();
		}
	}

	function send(msg) {
		appendToSendHistory(msg);
		appendMessage('send', msg);
		if (ws) {
			try {
				ws.send(msg);
			} catch (ex) {
				appendMessage('exception', 'Cannot send: ' + ex);
			}
		} else {
			appendMessage('exception', 'Cannot send: Not connected');
		}
	}

	function select(selector) {
		return document.querySelector(selector);
	}

	var maxSendHistorySize = 100;
		currentSendHistoryPosition = -1,
		sendHistoryRollback = '';

	function appendToSendHistory(msg) {
		currentSendHistoryPosition = -1;
		sendHistoryRollback = '';
		var sendHistory = JSON.parse(localStorage['websocketdconsole.sendhistory'] || '[]');
		if (sendHistory[0] !== msg) {
			sendHistory.unshift(msg);
			while (sendHistory.length > maxSendHistorySize) {
				sendHistory.pop();
			}
			localStorage['websocketdconsole.sendhistory'] = JSON.stringify(sendHistory);
		}
	}

	function moveThroughSendHistory(offset) {
		if (currentSendHistoryPosition == -1) {
			sendHistoryRollback = select('.send-input').value;
		}
		var sendHistory = JSON.parse(localStorage['websocketdconsole.sendhistory'] || '[]');
		currentSendHistoryPosition += offset;
		currentSendHistoryPosition = Math.max(-1, Math.min(sendHistory.length - 1, currentSendHistoryPosition));

		var el = select('.send-input');
		el.value = currentSendHistoryPosition == -1
			? sendHistoryRollback
			: sendHistory[currentSendHistoryPosition];
		setTimeout(function() {
			el.setSelectionRange(el.value.length, el.value.length);
		}, 0);
	}

	document.addEventListener("DOMContentLoaded", ready, false);

</script>

`
)

var ConsoleContent = defaultConsoleContent
