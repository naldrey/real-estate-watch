# real-estate-watch

Watches Fastmail for property-listing alert emails from Spanish real estate
portals, parses out new listings, and pushes them to Telegram. Single Go
binary, SQLite for storage, no other services.

```
IMAP (Fastmail) → per-portal parser → dedupe (SQLite) → Telegram
```

Saved-search alerts on each portal are delivered to their own Fastmail
folder (via a `+alias` and a filter rule). The watcher polls those folders,
parses new messages with each portal's own parser, dedupes against SQLite by
`(portal, external_id)`, and notifies on anything new. Telegram is optional;
without it the watcher still runs and logs. If polling starts failing (bad
credentials, network issues), you get one alert on the failure and one on
recovery, not one per poll.

### Supported portals

| Portal | Format |
|---|---|
| [idealista.com](https://www.idealista.com) | `text/html`, one listing per alert |
| [fotocasa.es](https://www.fotocasa.es) | `text/plain` part of a multipart alert |
| [habitaclia.com](https://www.habitaclia.com) | `text/html`, `iso-8859-1`, multiple listings per digest |
| [yaencontre.com](https://www.yaencontre.com) | `text/html`, multiple listings per digest |
| pisos.com | not implemented yet |

## Setup

**1. Fastmail.** Create an [app password](https://www.fastmail.help/hc/en-us/articles/360058752854)
scoped to IMAP (don't use your main password). For each portal, point its
alert emails at a `+alias` of your address, then add a Fastmail rule filing
each alias into a folder named exactly after the portal (`idealista`,
`fotocasa`, `habitaclia`, `yaencontre`). Unsure of a folder's real name?
Build and run `cmd/listmailboxes` to list them all.

**2. Telegram (optional).** Create a bot via [@BotFather](https://t.me/BotFather)
with `/newbot`. Message the bot once, then open
`https://api.telegram.org/bot<TOKEN>/getUpdates` to find your chat id in
`result[0].message.chat.id`.

**3. Configure.**

```bash
cp .env.example .env
```

Fill in `.env` (see [.env.example](.env.example) for all variables).
`IMAP_USERNAME` and `IMAP_APP_PASSWORD` are required, the rest have
defaults. `.env` is gitignored and loaded automatically; real environment
variables always win over it.

**4. Build and run.** Requires Go 1.26+.

```bash
go build -o watcher ./cmd/watcher
./watcher
```

Polls immediately, then every `POLL_INTERVAL` (default `5m`), until
`Ctrl+C`/`SIGTERM`. Structured JSON logs on stdout.

```bash
sqlite3 real-estate-watch.db "select portal, external_id, price_eur, rooms, size_m2, title from listings;"
```

**5. Run it continuously (macOS).** Use a `launchd` LaunchAgent:
`RunAtLoad`, `KeepAlive`, `WorkingDirectory` set to this repo, and a
`ThrottleInterval` so a crash loop doesn't hammer IMAP/Telegram.

```bash
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/<label>.plist   # start
launchctl kickstart -k gui/$(id -u)/<label>                            # restart after rebuild
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/<label>.plist     # stop
```

## Project layout

```
cmd/watcher/            entry point: config, store, IMAP client, poll loop
cmd/listmailboxes/      debug tool: lists every IMAP folder in the account

internal/listing/       domain model (Listing, dedupe Key, URL/price parsing)
internal/store/         SQLite persistence
internal/mail/          IMAP client (github.com/emersion/go-imap/v2)
internal/mimepart/      MIME helpers: HTML/plain-text extraction, charset decoding, Subject header
internal/ingest/        orchestration: Provider/Store/Notifier interfaces, parser registry, poll loop
internal/telegram/      Telegram Bot API client
internal/monitor/       poll health tracking, alerts on failure/recovery
internal/config/        env var + .env loading

internal/portal/<name>/ one package per portal: parser.go + testdata/*.eml
```

## Adding a new portal

1. Get a real sample alert email (`.eml`, "Show Original" in webmail).
2. Add `internal/portal/<name>/parser.go` implementing `Portal() string` and
   `Parse(body []byte) ([]listing.Listing, error)`, self-registering with
   `ingest.Register(parser{})` in `init()`.
3. Use `internal/mimepart` for extraction (`ExtractHTML`, `ExtractPlainText`,
   `Subject`). Don't hand-parse MIME/quoted-printable/charsets.
4. Add `testdata/*.eml` fixtures and a parser test.
5. Blank-import the new package in `cmd/watcher/main.go`.

## Testing

```bash
go build ./... && go vet ./... && go test ./...
```

Every parser is tested against real sample emails in `testdata/`, no live
IMAP or network needed.

## Conventions

- Errors: lowercase, no trailing punctuation, wrapped with `%w`.
- Structured logging via `log/slog`.
- Secrets from environment (or `.env`) only, never committed or hardcoded.
- Prices stored as integer euros, never floats.

## License

[MIT](LICENSE)
