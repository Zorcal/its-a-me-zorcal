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
	let suppressDisplayUpdate = false; // Flag to prevent display updates

	// Track locally stored newline commands for later server sync
	let pendingNewlines = 0;

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
		if (suppressDisplayUpdate) return;
		
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

	// Prevent empty command submissions to save number of HTTP requests
	const form = document.getElementById("command-form");
	form.addEventListener(
		"submit",
		(e) => {
			const command = actualInputValue.trim();
			if (command === "") {
				e.preventDefault();
				e.stopPropagation(); // Prevent HTMX from processing this event

				// Simulate empty command behavior locally without HTTP request
				const historyDiv = document.getElementById("command-history");
				const currentPrompt = document.getElementById("prompt").textContent;
				const emptyEntry = document.createElement("div");
				emptyEntry.innerHTML = `
				<div class="command-prompt">${currentPrompt}</div>
				<div class="command-output"></div>
			`;
				document.getElementById("command-output").appendChild(emptyEntry);

				// Track this newline for later server sync
				pendingNewlines++;

				// Reset input
				actualInputValue = "";
				input.value = "";
				input.setSelectionRange(0, 0);
				updateDisplay();

				// Scroll to bottom
				historyDiv.scrollTop = historyDiv.scrollHeight;

				// Focus back on input
				input.focus();
			} else {
				// Include any pending newlines as a parameter with the command
				input.value = actualInputValue;
				if (pendingNewlines > 0) {
					// Add hidden input for newlines
					const form = document.getElementById("command-form");
					let newlineInput = form.querySelector('input[name="newlines"]');
					if (!newlineInput) {
						newlineInput = document.createElement("input");
						newlineInput.type = "hidden";
						newlineInput.name = "newlines";
						form.appendChild(newlineInput);
					}
					newlineInput.value = pendingNewlines;
					pendingNewlines = 0;
				}
				// Let the normal command proceed (with newlines included)
			}
		},
		true,
	); // Use capture phase to run before HTMX

	// Handle keyboard shortcuts
	document.addEventListener("keydown", (e) => {
		// Only apply special handling when the input field is focused
		if (document.activeElement !== input) return;

		if (e.ctrlKey && e.key === "l") {
			e.preventDefault();
			// Suppress display updates to prevent flicker
			suppressDisplayUpdate = true;
			// Set the command but don't show it in the UI
			actualInputValue = "clear";
			input.value = "clear";
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

		// Reset everything after submission (only called for successful server requests)
		actualInputValue = "";
		form.reset();
		
		// Remove any hidden newline input
		const newlineInput = form.querySelector('input[name="newlines"]');
		if (newlineInput) {
			newlineInput.remove();
		}
		
		document.getElementById("command-input").focus();
		document.getElementById("input-text").textContent = "";
		document.getElementById("command-history").scrollTop =
			document.getElementById("command-history").scrollHeight;

		// Re-enable display updates and update display to show cursor at beginning
		suppressDisplayUpdate = false;
		updateDisplay();

		// Refresh command history immediately - the command response has already been processed
		fetchCommandHistory();
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

	// Store pending newlines before page unload
	window.addEventListener("beforeunload", () => {
		if (pendingNewlines > 0) {
			// Store in localStorage to handle on next page load
			localStorage.setItem('pendingNewlines', pendingNewlines.toString());
		}
	});

	// Handle any pending newlines from previous session (after all event listeners are set up)
	const storedNewlines = localStorage.getItem('pendingNewlines');
	if (storedNewlines) {
		pendingNewlines = parseInt(storedNewlines) || 0;
		localStorage.removeItem('pendingNewlines');
		
		// Send stored newlines immediately if any exist
		if (pendingNewlines > 0) {
			const params = new URLSearchParams();
			params.append("count", pendingNewlines);
			fetch("/newline", {
				method: "POST",
				headers: {
					"Content-Type": "application/x-www-form-urlencoded",
				},
				body: params,
			}).finally(() => {
				pendingNewlines = 0;
				// Refresh the page to show the newlines
				window.location.reload();
			});
		}
	}
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
