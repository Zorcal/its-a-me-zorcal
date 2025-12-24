document.addEventListener("DOMContentLoaded", () => {
	// Check screen size and show warning if needed
	checkScreenSize();
	window.addEventListener('resize', checkScreenSize);

	const cursor = document.getElementById("cursor");
	const input = document.getElementById("command-input");
	const inputText = document.getElementById("input-text");
	const prompt = document.getElementById("prompt");

	// Blink cursor
	setInterval(() => {
		cursor.style.opacity = cursor.style.opacity === "0" ? "1" : "0";
	}, 500);

	// Mirror input text
	input.addEventListener("input", () => {
		inputText.textContent = input.value;
	});

	// Handle Ctrl+L
	document.addEventListener("keydown", (e) => {
		if (e.ctrlKey && e.key === "l") {
			e.preventDefault();
			input.value = "clear";
			inputText.textContent = "clear";
			htmx.trigger("#command-form", "submit");
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
	};

	window.handleCommandError = function(event) {
		document.getElementById('command-output').innerHTML += event.detail.xhr.responseText;
		document.getElementById('command-history').scrollTop = document.getElementById('command-history').scrollHeight;
	};
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
