# real-estate-watch

A personal tool that watches Fastmail for new property-listing alert emails
from Spanish real estate portals, parses out the listings, and pushes new
ones to Telegram. Runs as a long-lived process that polls on an interval —
no external services, no database server, no containers, just a single Go
binary and a SQLite file.

## How it works

You set up saved-search alerts on each portal (idealista, fotocasa,
habitaclia, yaencontre, ...), each delivered to its own folder in your
Fastmail inbox (e.g. via a `+portalname` address and a filter rule). The
watcher polls those folders over IMAP, runs each portal's parser against new
messages, dedupes listings against a local SQLite database, and sends a
Telegram message for anything new.

```
IMAP (Fastmail) → per-portal parser → dedupe (SQLite) → Telegram
```

- **One mailbox folder per portal.** The watcher polls a folder named after
  each registered portal (e.g. `idealista`, `fotocasa`).
- **One parser per portal**, self-registering via `init()`. Each portal's
  alert emails have a completely different format — some are `text/html`,
  some are `text/plain`, some use different charsets — so each parser
  handles its own extraction.
- **Dedupe by `(portal, external_id)`.** A listing is never sent twice, even
  across restarts. Processed IMAP messages are also tracked by
  `(mailbox, uid)` so nothing gets re-fetched or re-parsed.
- **Telegram notifications are optional.** Without credentials configured,
  the watcher still runs and logs new listings; with them, it also pushes a
  message per new listing.

### Supported portals

| Portal | Format |
|---|---|
| [idealista.com](https://www.idealista.com) | `text/html`, one listing per alert |
| [fotocasa.es](https://www.fotocasa.es) | `text/plain` part of a multipart alert (much easier to parse than its HTML) |
| [habitaclia.com](https://www.habitaclia.com) | `text/html`, `iso-8859-1` charset, multiple listings per digest |
| [yaencontre.com](https://www.yaencontre.com) | `text/html`, multiple listings per digest |
| pisos.com | not implemented yet |

## Setup

### 1. Fastmail

1. Create an [app password](https://www.fastmail.help/hc/en-us/articles/360058752854) scoped to IMAP (Settings → Password & Security → App passwords). Don't use your main account password.
2. For each portal, set up a saved search / email alert on the portal's own
   site, sent to a `+alias` of your Fastmail address (e.g.
   `you+idealista@fastmail.com`).
3. In Fastmail, add a rule that files mail sent to each `+alias` into a
   folder named exactly after the portal (`idealista`, `fotocasa`,
   `habitaclia`, `yaencontre`). The watcher polls folders by that exact
   name.
4. Not sure what a folder is actually called, or whether the rule worked?
   Build and run `cmd/listmailboxes` (see below) to list every folder in the
   account.

### 2. Telegram (optional)

1. Create a bot via [@BotFather](https://t.me/BotFather): send `/newbot` and
   follow the prompts. It gives you a bot token.
2. Message your new bot anything (e.g. "hi"), then open
   `https://api.telegram.org/bot<TOKEN>/getUpdates` in a browser — your chat
   id is in `result[0].message.chat.id`.

### 3. Configure

```bash
cp .env.example .env
```

Edit `.env` with your real values. See [.env.example](.env.example) for the
full list of variables; `IMAP_USERNAME` and `IMAP_APP_PASSWORD` are
required, everything else is optional with sensible defaults.

`.env` is gitignored and loaded automatically at startup — no need to
`export` anything by hand. Real environment variables, if set, always take
priority over `.env`.

### 4. Build and run

Requires Go 1.26+.

```bash
go build -o watcher ./cmd/watcher
go build -o listmailboxes ./cmd/listmailboxes  # optional debug tool
./watcher
```

The watcher polls immediately on startup, then again every `POLL_INTERVAL`
(default `5m`), until stopped with `Ctrl+C` / `SIGTERM`. Logs are structured
JSON on stdout.

Inspect what's been saved:

```bash
sqlite3 real-estate-watch.db "select portal, external_id, price_eur, rooms, size_m2, title from listings;"
```

### 5. Run it continuously (macOS)

To keep it running in the background across logins and auto-restart on
crash, use a `launchd` LaunchAgent. See the plist template pattern in
`~/Library/LaunchAgents/` — key settings: `RunAtLoad`, `KeepAlive`, a
`WorkingDirectory` set to this repo (so `.env` and the SQLite file resolve),
and `ThrottleInterval` to avoid a crash loop hammering IMAP/Telegram.

```bash
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/<label>.plist   # start
launchctl kickstart -k gui/$(id -u)/<label>                            # restart after rebuilding
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/<label>.plist     # stop
```

## Project layout

```
cmd/watcher/          entry point: config → store → IMAP client → poll loop
cmd/listmailboxes/     debug tool: lists every IMAP folder in the account

internal/listing/      domain model (Listing, dedupe Key, URL normalization)
internal/store/        SQLite persistence (listings + processed-message tracking)
internal/mail/         IMAP client (github.com/emersion/go-imap/v2)
internal/mimepart/      MIME helpers shared by every parser: HTML/plain-text
                        extraction, charset decoding, Subject header reading
internal/ingest/       orchestration: Provider/Store/Notifier interfaces,
                        MessageParser interface + registry, the poll-and-parse loop
internal/telegram/      Telegram Bot API client (implements ingest.Notifier)
internal/config/       env var + .env loading

internal/portal/<name>/ one package per portal, each with its own
                        parser.go + testdata/*.eml fixtures from real emails
```

## Adding a new portal

1. Get a real sample alert email (`.eml`, via "Show Original" / "View
   source" in webmail).
2. Add `internal/portal/<name>/parser.go`: implement `Portal() string` and
   `Parse(body []byte) ([]listing.Listing, error)`, and self-register with
   `ingest.Register(parser{})` in `init()`.
3. Use `internal/mimepart` to extract the relevant part (`ExtractHTML`,
   `ExtractPlainText`, or `Subject` for filtering by subject line) — don't
   parse MIME/quoted-printable/charsets by hand.
4. Add `testdata/*.eml` fixtures (trim the verbose routing/auth headers,
   keep the real body) and a parser test asserting the extracted fields.
5. Blank-import the new package in `cmd/watcher/main.go`.

## Testing

```bash
go build ./... && go vet ./... && go test ./...
```

Every parser is tested against real (header-trimmed) sample emails in its
`testdata/` directory — no live IMAP or network access needed to run the
test suite.

## Conventions

- All code, comments, logs, and errors in English.
- Errors: lowercase, no trailing punctuation, wrapped with `%w`.
- Structured logging via `log/slog`.
- Secrets from environment (or `.env`) only — never committed, never hardcoded.
- Prices are stored as integer euros, never floats.

## License

[MIT](LICENSE)
