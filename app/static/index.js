document.addEventListener("DOMContentLoaded", () => {
	// Check screen size and show warning if needed
	checkScreenSize();
	window.addEventListener("resize", checkScreenSize);

	const input = document.getElementById("command-input");
	const inputText = document.getElementById("input-text");
	const prompt = document.getElementById("prompt");

	// Command history navigation
	let commandHistory = [];
	let historyIndex = -1;

	// Text-based cursor management
	let actualInputValue = ""; // The real command without the cursor

	// Fetch command history from server
	async function fetchCommandHistory() {
		try {
			const response = await fetch("/history");
			if (response.ok) {
				const data = await response.json();
				commandHistory = Array.isArray(data) ? data : [];
				historyIndex = commandHistory.length; // Start at end of history
			} else {
				console.warn("Failed to fetch command history, using empty array");
				commandHistory = [];
				historyIndex = 0;
			}
		} catch (error) {
			console.error("Failed to fetch command history:", error);
			commandHistory = []; // Always ensure it's an array
			historyIndex = 0;
		}
	}

	// Navigate command history
	function navigateHistory(direction) {
		if (!commandHistory || commandHistory.length === 0) return;

		if (direction === "up") {
			if (historyIndex > 0) {
				historyIndex--;
				const newValue = commandHistory[historyIndex];
				actualInputValue = newValue;
				// Move cursor to end of command first
				input.value = actualInputValue;
				input.setSelectionRange(newValue.length, newValue.length);
				updateDisplay();
			}
		} else if (direction === "down") {
			if (historyIndex < commandHistory.length - 1) {
				historyIndex++;
				const newValue = commandHistory[historyIndex];
				actualInputValue = newValue;
				// Move cursor to end of command first
				input.value = actualInputValue;
				input.setSelectionRange(newValue.length, newValue.length);
				updateDisplay();
			} else if (historyIndex === commandHistory.length - 1) {
				historyIndex = commandHistory.length;
				actualInputValue = "";
				// Reset cursor to beginning for empty input
				input.value = actualInputValue;
				input.setSelectionRange(0, 0);
				updateDisplay();
			}
		}
	}

	// Initial fetch of command history
	fetchCommandHistory();

	// Update display with cursor at current position
	function updateDisplay() {
		const cursorPos = input.selectionStart || 0;
		const beforeCursor = actualInputValue.substring(0, cursorPos);
		const afterCursor = actualInputValue.substring(cursorPos);

		// Direct assignment for fastest update
		inputText.textContent = beforeCursor + "│" + afterCursor;

		// Only sync input value if it's different (avoid unnecessary DOM updates)
		if (input.value !== actualInputValue) {
			const savedPos = input.selectionStart;
			input.value = actualInputValue;
			input.setSelectionRange(savedPos, savedPos);
		}
	}

	input.addEventListener("input", () => {
		actualInputValue = input.value;
		updateDisplay();
	});

	// Update display when cursor moves
	input.addEventListener("keydown", (e) => {
		if (
			e.key === "ArrowLeft" ||
			e.key === "ArrowRight" ||
			e.key === "Home" ||
			e.key === "End"
		) {
			requestAnimationFrame(updateDisplay);
		}
	});

	input.addEventListener("click", updateDisplay);
	input.addEventListener("mouseup", updateDisplay);
	input.addEventListener("selectionchange", updateDisplay);

	// Initial display
	updateDisplay();

	// Handle keyboard shortcuts
	document.addEventListener("keydown", (e) => {
		// Only apply special handling when the input field is focused
		if (document.activeElement !== input) return;

		if (e.ctrlKey && e.key === "l") {
			e.preventDefault();
			input.value = "clear";
			inputText.textContent = "clear";
			htmx.trigger("#command-form", "submit");
		}

		if (e.ctrlKey && e.key === "c") {
			e.preventDefault();
			// Clear current input
			actualInputValue = "";
			input.value = "";
			input.setSelectionRange(0, 0);
			updateDisplay();
		}

		// Command history navigation with up/down arrow keys
		if (e.key === "ArrowUp") {
			e.preventDefault();
			navigateHistory("up");
		}
		if (e.key === "ArrowDown") {
			e.preventDefault();
			navigateHistory("down");
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
	window.handleCommandSubmit = () => {
		const form = document.getElementById("command-form");
		const submittedCommand = actualInputValue.trim();

		// Ensure we submit the actual command without cursor
		input.value = actualInputValue;

		// Reset everything after submission
		actualInputValue = "";
		form.reset();
		document.getElementById("command-input").focus();
		document.getElementById("input-text").textContent = "";
		document.getElementById("command-history").scrollTop =
			document.getElementById("command-history").scrollHeight;

		// Update display to show cursor at beginning
		updateDisplay();

		if (submittedCommand !== "") {
			fetchCommandHistory();
		}
	};

	window.handleCommandError = (event) => {
		document.getElementById("command-output").innerHTML +=
			event.detail.xhr.responseText;
		document.getElementById("command-history").scrollTop =
			document.getElementById("command-history").scrollHeight;
	};

	// Handle open command response - check for X-Open-URL header and open new tab
	document.body.addEventListener("htmx:afterRequest", (event) => {
		const xhr = event.detail.xhr;
		const openUrl = xhr.getResponseHeader("X-Open-URL");
		if (openUrl) {
			window.open(openUrl, "_blank");
		}
	});
});

function checkScreenSize() {
	const isSmallScreen = window.innerWidth <= 768 || window.innerHeight <= 600;
	let warning = document.getElementById("screen-size-warning");

	if (isSmallScreen && !warning) {
		warning = document.createElement("div");
		warning.id = "screen-size-warning";
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
