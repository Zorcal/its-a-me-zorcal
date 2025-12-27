document.addEventListener("DOMContentLoaded", () => {
	// Check screen size and show warning if needed
	checkScreenSize();
	window.addEventListener('resize', checkScreenSize);

	const cursor = document.getElementById("cursor");
	const input = document.getElementById("command-input");
	const inputText = document.getElementById("input-text");
	const prompt = document.getElementById("prompt");
	
	// Command history navigation
	let commandHistory = [];
	let historyIndex = -1;
	
	// Fetch command history from server
	async function fetchCommandHistory() {
		try {
			const response = await fetch('/history');
			if (response.ok) {
				commandHistory = await response.json();
				historyIndex = commandHistory.length; // Start at end of history
			}
		} catch (error) {
			console.error('Failed to fetch command history:', error);
		}
	}
	
	// Navigate command history
	function navigateHistory(direction) {
		if (commandHistory.length === 0) return;
		
		if (direction === 'up') {
			if (historyIndex > 0) {
				historyIndex--;
				input.value = commandHistory[historyIndex];
				inputText.textContent = commandHistory[historyIndex];
			}
		} else if (direction === 'down') {
			if (historyIndex < commandHistory.length - 1) {
				historyIndex++;
				input.value = commandHistory[historyIndex];
				inputText.textContent = commandHistory[historyIndex];
			} else if (historyIndex === commandHistory.length - 1) {
				historyIndex = commandHistory.length;
				input.value = "";
				inputText.textContent = "";
			}
		}
	}
	
	// Initial fetch of command history
	fetchCommandHistory();

	// Blink cursor
	setInterval(() => {
		cursor.style.opacity = cursor.style.opacity === "0" ? "1" : "0";
	}, 500);

	// Mirror input text
	input.addEventListener("input", () => {
		inputText.textContent = input.value;
	});

	// Handle keyboard shortcuts
	document.addEventListener("keydown", (e) => {
		if (e.ctrlKey && e.key === "l") {
			e.preventDefault();
			input.value = "clear";
			inputText.textContent = "clear";
			htmx.trigger("#command-form", "submit");
		}
		
		// Command history navigation with arrow keys
		if (e.key === "ArrowUp") {
			e.preventDefault();
			navigateHistory('up');
		}
		
		if (e.key === "ArrowDown") {
			e.preventDefault();
			navigateHistory('down');
		}
		
		// Prevent default tab behavior (common in terminals for autocomplete)
		if (e.key === "Tab") {
			e.preventDefault();
		}
	});

	// Update prompt when server sends trigger
	document.body.addEventListener("updatePrompt", (e) => {
		prompt.textContent = e.detail.updatePrompt;
	});

	// Focus input on click anywhere
	document.addEventListener("click", () => {
		input.focus();
	});

	// HTMX event handlers
	window.handleCommandSubmit = function() {
		const form = document.getElementById('command-form');
		form.reset();
		document.getElementById('command-input').focus();
		document.getElementById('input-text').textContent = '';
		document.getElementById('command-history').scrollTop = document.getElementById('command-history').scrollHeight;
		
		// Refresh command history after submission
		fetchCommandHistory();
	};

	window.handleCommandError = function(event) {
		document.getElementById('command-output').innerHTML += event.detail.xhr.responseText;
		document.getElementById('command-history').scrollTop = document.getElementById('command-history').scrollHeight;
	};

	// Handle open command response - check for X-Open-URL header and open new tab
	document.body.addEventListener('htmx:afterRequest', function(event) {
		const xhr = event.detail.xhr;
		const openUrl = xhr.getResponseHeader('X-Open-URL');
		if (openUrl) {
			window.open(openUrl, '_blank');
		}
	});
});

function checkScreenSize() {
	const isSmallScreen = window.innerWidth <= 768 || window.innerHeight <= 600;
	let warning = document.getElementById('screen-size-warning');
	
	if (isSmallScreen && !warning) {
		warning = document.createElement('div');
		warning.id = 'screen-size-warning';
		warning.innerHTML = `
			<div class="warning-content">
				<h2>Screen Resolution Not Supported</h2>
				<p>This terminal interface requires a larger screen.</p>
				<p>Please use a desktop or tablet device.</p>
			</div>
		`;
		document.body.appendChild(warning);
	} else if (!isSmallScreen && warning) {
		warning.remove();
	}
}
