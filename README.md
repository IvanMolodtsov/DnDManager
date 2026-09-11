# DnDManager

Local D&D 5e (2014) campaign and character manager for a small table. Create a campaign, invite players, run the character wizard, then level up.

**Stack:** Go, SQLite (`modernc.org/sqlite`), HTML templates + vendored HTMX. Styles come from the compiled Tailwind stylesheet in `web/static/css/app.css` (preflight, theme, utilities, and custom components). No Node build.

## Run

From the repo root (Go 1.24+):

```bash
go run ./cmd/web
```

Windows: `scripts/start.ps1`. Unix: `scripts/start.sh`. Tests: `go test ./...` or `scripts/test.ps1`.

| Env | Default | |
| --- | --- | --- |
| `ADDR` | `:8080` | Listen address |
| `DATA_DIR` | `data` | SQLite directory (`dnd.db`) |

Open http://localhost:8080. The first registered user is **Admin**; later accounts are players.

## Using it

- **Campaigns:** creator is the Dungeon Master. Share the invite code; others join as players.
- **Characters:** wizard (name/race/background → class → standard array → confirm), then level-up. The campaign DM can view sheets read-only.
- **Catalog:** PHB 5e (2014) seed. Russian source links use `https://5e14.dnd.su/{type}/{numericId}-{english-slug}/` (`class`, `race`, `backgrounds`, `spells`, `items`). Subclass/feature rows link to the parent page when 5e14 has no dedicated URL.
- **Language:** EN/RU in the header (cookie + saved on the account).

`data/` is local and not in git. SQL under `migrations/` runs on startup — **restart the server** after pulling new migration files.
