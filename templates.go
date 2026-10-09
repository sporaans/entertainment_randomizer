package main

import "html/template"

var (
	homeTemplate  = template.Must(template.New("home").Parse(layoutStart + homeHTML + layoutEnd))
	setupTemplate = template.Must(template.New("setup").Parse(layoutStart + setupHTML + layoutEnd))
)

const layoutStart = `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Spotify Randomizer</title>
	<style>
		:root {
			color-scheme: light;
			--bg: #f7f8f5;
			--panel: #ffffff;
			--ink: #172016;
			--muted: #687166;
			--line: #dfe5dc;
			--accent: #1db954;
			--accent-dark: #14833b;
			--warn-bg: #fff2d8;
			--warn-ink: #704700;
			--shadow: 0 22px 70px rgba(23, 32, 22, 0.10);
		}

		* { box-sizing: border-box; }

		body {
			margin: 0;
			min-height: 100vh;
			font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
			background:
				linear-gradient(130deg, rgba(29, 185, 84, 0.13), rgba(255, 255, 255, 0) 34%),
				radial-gradient(circle at 85% 12%, rgba(83, 122, 255, 0.10), transparent 30%),
				var(--bg);
			color: var(--ink);
		}

		main {
			width: min(980px, calc(100% - 32px));
			min-height: 100vh;
			margin: 0 auto;
			display: grid;
			align-content: center;
			gap: 24px;
			padding: 48px 0;
		}

		.shell {
			display: grid;
			grid-template-columns: minmax(0, 1fr) 320px;
			gap: 18px;
			align-items: stretch;
		}

		.hero, .side {
			background: color-mix(in srgb, var(--panel) 92%, transparent);
			border: 1px solid rgba(223, 229, 220, 0.9);
			border-radius: 8px;
			box-shadow: var(--shadow);
		}

		.hero { padding: clamp(28px, 5vw, 54px); }

		.side {
			padding: 24px;
			display: flex;
			flex-direction: column;
			justify-content: space-between;
			gap: 28px;
		}

		.kicker {
			margin: 0 0 18px;
			font-size: 0.78rem;
			font-weight: 700;
			letter-spacing: 0;
			color: var(--accent-dark);
			text-transform: uppercase;
		}

		h1 {
			max-width: 680px;
			margin: 0;
			font-size: clamp(2.5rem, 7vw, 5.6rem);
			line-height: 0.95;
			letter-spacing: 0;
		}

		h2 {
			margin: 0;
			font-size: clamp(1.7rem, 4vw, 2.7rem);
			line-height: 1.08;
			letter-spacing: 0;
		}

		.lede {
			max-width: 620px;
			margin: 24px 0 0;
			color: var(--muted);
			font-size: 1.08rem;
			line-height: 1.65;
		}

		.actions {
			display: flex;
			flex-wrap: wrap;
			gap: 12px;
			margin-top: 34px;
		}

		button, .button {
			appearance: none;
			border: 0;
			border-radius: 999px;
			display: inline-flex;
			align-items: center;
			justify-content: center;
			gap: 10px;
			min-height: 48px;
			padding: 0 20px;
			background: var(--ink);
			color: #fff;
			font: inherit;
			font-weight: 750;
			text-decoration: none;
			cursor: pointer;
			transition: transform 150ms ease, background 150ms ease, box-shadow 150ms ease;
		}

		button:hover, .button:hover {
			transform: translateY(-1px);
			box-shadow: 0 12px 24px rgba(23, 32, 22, 0.14);
		}

		button:disabled {
			cursor: wait;
			opacity: 0.72;
			transform: none;
			box-shadow: none;
		}

		.button.primary, button.primary {
			background: var(--accent);
			color: #08210f;
		}

		.button.secondary {
			background: #eef2eb;
			color: var(--ink);
		}

		.status, .result, .setup-panel {
			border: 1px solid var(--line);
			border-radius: 8px;
			padding: 18px;
			background: #fbfcfa;
		}

		.status-title {
			margin: 0 0 6px;
			font-weight: 780;
		}

		.status-copy, .help-copy {
			margin: 0;
			color: var(--muted);
			line-height: 1.5;
		}

		.result {
			margin-top: 24px;
			padding: 24px;
			display: none;
		}

		.result.visible { display: block; }

		.result-label {
			margin: 0 0 10px;
			color: var(--accent-dark);
			font-size: 0.82rem;
			font-weight: 780;
		}

		.meta {
			display: grid;
			gap: 8px;
			margin-top: 20px;
			color: var(--muted);
		}

		.meta strong { color: var(--ink); }

		.meta a {
			color: var(--accent-dark);
			font-weight: 700;
		}

		.notice {
			display: none;
			margin-top: 16px;
			padding: 12px 14px;
			border-radius: 8px;
			background: var(--warn-bg);
			color: var(--warn-ink);
			font-weight: 650;
		}

		.notice.visible, .notice.static { display: block; }

		.dot {
			width: 10px;
			height: 10px;
			border-radius: 999px;
			background: var(--accent);
			box-shadow: 0 0 0 6px rgba(29, 185, 84, 0.15);
		}

		.connection {
			display: flex;
			align-items: center;
			gap: 12px;
			color: var(--muted);
			font-weight: 700;
		}

		.footer {
			color: var(--muted);
			font-size: 0.92rem;
			line-height: 1.5;
		}

		.movie-mode {
			padding: clamp(28px, 5vw, 54px);
			background: color-mix(in srgb, var(--panel) 92%, transparent);
			border: 1px solid rgba(223, 229, 220, 0.9);
			border-radius: 8px;
			box-shadow: var(--shadow);
		}

		.movie-mode h2 { max-width: 680px; }

		.movie-form {
			max-width: none;
		}

		.watchlist-row {
			display: grid;
			grid-template-columns: minmax(0, 1fr) auto;
			gap: 12px;
			align-items: end;
		}

		.movie-result h2 {
			font-size: clamp(1.8rem, 4vw, 3.5rem);
		}

		form {
			display: grid;
			gap: 16px;
			margin-top: 28px;
			max-width: 640px;
		}

		label {
			display: grid;
			gap: 8px;
			color: var(--muted);
			font-weight: 700;
		}

		input {
			width: 100%;
			min-height: 46px;
			border: 1px solid var(--line);
			border-radius: 8px;
			padding: 0 14px;
			background: #fff;
			color: var(--ink);
			font: inherit;
		}

		input:focus {
			border-color: var(--accent);
			box-shadow: 0 0 0 3px rgba(29, 185, 84, 0.14);
			outline: none;
		}

		.steps {
			display: grid;
			gap: 12px;
			margin: 0;
			padding-left: 20px;
			color: var(--muted);
			line-height: 1.5;
		}

		code {
			border: 1px solid var(--line);
			border-radius: 6px;
			padding: 2px 6px;
			background: #f4f6f2;
			color: var(--ink);
			font-size: 0.94em;
		}

		@media (max-width: 760px) {
			main {
				width: min(100% - 24px, 980px);
				align-content: start;
				padding: 20px 0;
			}

			.shell { grid-template-columns: 1fr; }

			.hero, .side { box-shadow: 0 14px 44px rgba(23, 32, 22, 0.08); }

			.watchlist-row { grid-template-columns: 1fr; }
		}
	</style>
</head>
<body>
	<main>
`

const homeHTML = `		<section class="shell" aria-label="Spotify Randomizer">
			<div class="hero">
				<p class="kicker">Spotify Randomizer</p>
				<h1>Pick an album without overthinking it.</h1>
				<p class="lede">Connect your Spotify account, then let the app choose one saved album from your library. Simple, local, and ready for a home server.</p>

				{{if .IsConnected}}
				<div class="actions">
					<button class="primary" id="pickButton" type="button">Pick random album</button>
					<a class="button secondary" href="/login?force=1">Reconnect Spotify</a>
					<a class="button secondary" href="/setup">Spotify client setup</a>
				</div>
				{{else}}
				<div class="actions">
					<a class="button primary" href="/login">Connect Spotify</a>
					<a class="button secondary" href="/setup">Spotify client setup</a>
				</div>
				{{end}}

				<div class="notice" id="notice" role="status"></div>

				<div class="result" id="result" aria-live="polite">
					<p class="result-label">Your random album</p>
					<h2 id="albumName"></h2>
					<div class="meta">
						<div><strong>Artist:</strong> <span id="albumArtists"></span></div>
						<div><strong>Release date:</strong> <span id="albumRelease"></span></div>
						<div><strong>Library size:</strong> <span id="albumTotal"></span> saved albums</div>
						<div><a id="albumLink" href="#" rel="noreferrer" target="_blank">Open in Spotify</a></div>
					</div>
				</div>
			</div>

			<aside class="side">
				<div class="status">
					<p class="status-title">Status</p>
					<p class="status-copy">
						{{if .IsConnected}}Spotify is connected and ready.{{else}}Spotify needs to be connected before picking an album.{{end}}
					</p>
				</div>

				<div class="connection">
					<span class="dot" aria-hidden="true"></span>
					<span>{{if .IsConnected}}Connected{{else}}Not connected{{end}}</span>
				</div>

				<p class="footer">Redirect URL: <code>{{.Settings.RedirectURL}}</code></p>
			</aside>
		</section>

		<section class="movie-mode" aria-label="IMDb Watchlist Randomizer">
			<p class="kicker">IMDb Watchlist Randomizer</p>
			<h2>Pick a movie from a public watchlist.</h2>
			<p class="lede">Paste a public IMDb watchlist link and let the app choose one title at random.</p>

			<form class="movie-form" id="movieForm">
				<div class="watchlist-row">
					<label>
						Public IMDb watchlist link
						<input id="watchlistURL" name="watchlist_url" type="url" required
							placeholder="https://www.imdb.com/user/ur12345678/watchlist/"
							autocomplete="url">
					</label>
					<button class="primary" id="movieButton" type="submit">Pick random movie</button>
				</div>
			</form>

			<div class="notice" id="movieNotice" role="status"></div>

			<div class="result movie-result" id="movieResult" aria-live="polite">
				<p class="result-label">Your random movie</p>
				<h2 id="movieName"></h2>
			</div>
		</section>

		<script>
			const button = document.querySelector("#pickButton");
			const notice = document.querySelector("#notice");
			const result = document.querySelector("#result");
			const movieForm = document.querySelector("#movieForm");
			const movieButton = document.querySelector("#movieButton");
			const movieNotice = document.querySelector("#movieNotice");
			const movieResult = document.querySelector("#movieResult");

			function showNotice(message) {
				notice.textContent = message;
				notice.classList.add("visible");
			}

			function clearNotice() {
				notice.textContent = "";
				notice.classList.remove("visible");
			}

			if (button) {
				button.addEventListener("click", async () => {
					clearNotice();
					button.disabled = true;
					button.textContent = "Picking...";

					try {
						const response = await fetch("/api/random-album", { method: "POST" });
						const payload = await response.json();

						if (!response.ok) {
							throw new Error(payload.error || "Could not pick an album.");
						}

						document.querySelector("#albumName").textContent = payload.name;
						document.querySelector("#albumArtists").textContent = payload.artists;
						document.querySelector("#albumRelease").textContent = payload.releaseDate || "Unknown";
						document.querySelector("#albumTotal").textContent = payload.total;
						document.querySelector("#albumLink").href = payload.spotifyUrl;
						result.classList.add("visible");
					} catch (error) {
						showNotice(error.message);
					} finally {
						button.disabled = false;
						button.textContent = "Pick random album";
					}
				});
			}

			movieForm.addEventListener("submit", async (event) => {
				event.preventDefault();
				movieNotice.textContent = "";
				movieNotice.classList.remove("visible");
				movieResult.classList.remove("visible");
				movieButton.disabled = true;
				movieButton.textContent = "Picking...";

				try {
					const response = await fetch("/api/random-movie", {
						method: "POST",
						headers: { "Content-Type": "application/json" },
						body: JSON.stringify({ url: document.querySelector("#watchlistURL").value })
					});
					const payload = await response.json();

					if (!response.ok) {
						throw new Error(payload.error || "Could not pick a movie.");
					}

					document.querySelector("#movieName").textContent = payload.name;
					movieResult.classList.add("visible");
				} catch (error) {
					movieNotice.textContent = error.message;
					movieNotice.classList.add("visible");
				} finally {
					movieButton.disabled = false;
					movieButton.textContent = "Pick random movie";
				}
			});
		</script>
`

const setupHTML = `		<section class="shell" aria-label="Spotify Client Setup">
			<div class="hero">
				<p class="kicker">First-run setup</p>
				<h1>Register your Spotify client.</h1>
				<p class="lede">Create a Spotify app, paste its client credentials here, and this app will save them locally for future runs.</p>

				{{if .Error}}
				<div class="notice static" role="alert">{{.Error}}</div>
				{{end}}

				<form method="post" action="/setup">
					<label>
						Client ID
						<input name="client_id" autocomplete="off" required value="{{.Settings.ClientID}}">
					</label>
					<label>
						Client secret
						<input name="client_secret" autocomplete="off" required type="password" value="{{.Settings.ClientSecret}}">
					</label>
					<label>
						Redirect URL
						<input name="redirect_url" required value="{{.Settings.RedirectURL}}">
					</label>

					<div class="actions">
						<button class="primary" type="submit">Save and connect Spotify</button>
						{{if .Settings.IsConfigured}}<a class="button secondary" href="/">Back</a>{{end}}
					</div>
				</form>
			</div>

			<aside class="side">
				<div class="setup-panel">
					<p class="status-title">Spotify dashboard steps</p>
					<ol class="steps">
						<li>Open <a href="https://developer.spotify.com/dashboard" rel="noreferrer" target="_blank">developer.spotify.com/dashboard</a>.</li>
						<li>Create an app, or open an existing app.</li>
						<li>Copy the Client ID and Client secret into this form.</li>
						<li>In app settings, add <code>{{.Settings.RedirectURL}}</code> as a redirect URI.</li>
						<li>Save the Spotify app settings, then submit this form.</li>
					</ol>
				</div>

				<p class="footer">Saved config file: <code>{{.ConfigPath}}</code>. Environment variables still work and override the saved setup.</p>
			</aside>
		</section>
`

const layoutEnd = `	</main>
</body>
</html>`
