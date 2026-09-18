# DnDManager — agent context

Local D&D 5e **2014** campaign/character manager for ~5 people. Owner: **Ivan**. Repo: https://github.com/IvanMolodtsov/DnDManager. Default branch: **main** (no `master`).

Keep this file and `.cursor/rules/project-context.mdc` in sync when product behavior changes.

## Stack

- Go + SQLite (`modernc.org/sqlite`). HTMX vendored. EN/RU i18n. **No Node.**
- Tailwind is **compiled** into `web/static/css/app.css`. **Do not use the Tailwind Play CDN** — it replaced the compiled sheet and broke the look. Append small extras at the end of `app.css` if needed.
- Run from repo root: `./start.ps1` / `./start.sh` (Air hot reload, `.air.toml`). One-shot: `scripts/start-once.ps1` / `scripts/start-once.sh` or `go run ./cmd/web`. Tests: `go test ./...`.
- Air rebuilds on `.go` / `.html` / `.json` / `.sql` / `.css`. New SQL runs on process start (Air restart is enough).
- **SQLite MCP** (project `.cursor/mcp.json`, server `dndmanager-db`): read-only `list_tables` / `describe_table` / `query` against `data/dnd.db`. Use it for live campaign data. Reload MCP in Cursor if the server is red. Do not commit `data/`.

## Layout

Spring-style **DTO / Service / Repository / Controller** under `internal/{platform,users,campaigns,catalog,rules,characters}` plus stubs `items` / `abilities`. Templates in `web/templates`.

Sheet HTMX islands: `#sheet-stats` (abilities/skills), `#sheet-magic` (resources/spells), `#sheet-vitals` (defense + statuses), `#sheet-inventory` (equipped / pack / consumables). Spell/check/weapon-attack modal is `#spell-use-modal`. HP/temp damage and heal modal is `#hp-adjust-modal`. Item add wizard is `#item-add-modal`. Casting a spell or equipping/using an item swaps the island and OOB-updates `#sheet-vitals` (equip also OOB-updates `#sheet-stats` and `#sheet-magic` for grants). After HP/temp apply, the submitter swaps from the POST (OOB `#sheet-vitals`); other viewers with `CanView` refresh `#sheet-vitals` over **SSE** (`GET /characters/{id}/events`, EventSource → `GET /characters/{id}/vitals`).

## Access

- Open registration. First registered user is **Admin**.
- **DM is campaign-scoped**: the campaign creator is DM of that campaign (not a global role).
- Join via invite codes.
- Players edit their own characters. Campaign DM may edit HP, temp HP, death saves, and remove statuses; other sheet mutations stay owner-only. Fellow players cannot edit.

## Character wizard

Draft until confirm:

1. Name / race / background
2. Class / subclass (RAW: subclass only when class level grants it)
3. Assign **15, 14, 13, 12, 10, 8** plus racial bonuses
4. Confirm

## Level-up

- Choose class each level (multiclass OK).
- HP: average or IRL roll (1–hit die).
- ASI is **+2 only** (feat picker is **TODO**).
- Proficiency bonus from **total** character level, not class level.
- Keep subclass `<select>` **outside** the HTMX preview swap (it used to reset to “choose…”).

## Catalog

- Canonical URLs: `https://5e14.dnd.su/{type}/{id}-{slug}/` (e.g. `/class/104-warlock/`).
- Discover IDs from 5e14 indexes; **never invent** paths like `/class/warlock/`.
- Use 5eapi `/api/2014/` when it returns 200.
- **No scraping article body.**
- Seed **PHB classes only** (no Artificer, sidekicks, or homebrew). ~378 spells.
- Monsters: `scripts/genmonsters` writes `014`. 5eapi 2014 monsters (fightable HP/AC/CR/DEX/resist/actions) plus 5e14 URLs when names match. 5e14-only cards (Spiderdragon, adventure NPCs, MotM, …) are **stubs**: name_en/ru, slug, canonical 5e14 URL, CR/type from the **index/filter JSON** (`piece/bestiary/index-list` + `/bestiary/` listing maps). `is_stub=1`, HP 0. **No article-body scrape.** WotC sources included; skip a huge homebrew dump if that group appears, but always keep Spiderdragon (and similar unique names). IDs: 5e14 numeric when matched or stubbed; 5eapi-only from `1000000` by slug sort. Regen: `go run ./scripts/genmonsters` from repo root (needs network). Search is Unicode case-insensitive; filter danger = CR; SRD rows sort before stubs. Stub add uses 1 HP so the DM can HP-edit immediately. Changing `014` does not reseed an already-migrated `data/dnd.db` — re-run that file’s DELETE+INSERT (and `ALTER TABLE catalog_monsters ADD COLUMN is_stub …` if the column is missing). Battle units snapshot names/HP, so live fights stay intact.
- Items: `scripts/genitems` writes `010`. 5eapi 2014 equipment + magic-items (stats + SRD `desc`) plus 5e14 **DMG stubs** (Moonblade etc.) that need manual configure. Filter 5e14 to WotC DMG (source 101); skip homebrew. IDs: 5e14 numeric when matched; 5eapi-only from `1000000` by slug sort. **No scraping article body.** Regen: `go run ./scripts/genitems` from repo root (needs network).
- PHB arms: migration `012` flags ~37 bases (`is_base`) and overlays RU names / PHB properties / the arms-table URL. Structured table in `internal/catalog/phb_arms.go` (same idea as `scripts/genspells/formulas.go`). Generic one-stat defs live in `catalog_stat_features` (class/race grants stay in `catalog_features`).
- PHB armor: migration `013` flags 12 suits + shield as `is_base` and overlays RU names / PHB AC / `https://5e14.dnd.su/articles/inventory/95-armor-and-shields/`. Table in `internal/catalog/phb_armor.go`. Jewelry has no PHB table — slot bases `ring` / `neck` / `cloak` / `head` / `gloves` / `boots` / `belt` (`kind=jewelry`, ids `2000001+`).
- Parameterized pickers and DMG artifact templates are seeded in `013` (`catalog_stat_features`). Mechanics encoded in `internal/catalog/dmg_artifacts.go` (link `https://5e14.dnd.su/articles/inventory/139-artifacts/`; **no HTML scrape**, no d100 roller).
- Many PHB spells have **empty 5eapi damage** (Hex, Witch Bolt, Chromatic Orb, …). Overlay in `scripts/genspells/formulas.go` + migration `006`. Regen of `004` should call `applyPHBFormula`. Do not scrape dnd.su lore for dice.

## Spells UI

- Prepared list on the sheet (plus **virtual at-will grants** from equipped items, badge “from {item}”); learned list while editing. Grants are **not** written to `character_spells`.
- **Use** opens a popup with the **catalog formula** (IRL roll) and **Auto-calculate** (server `RollFormula`). **Cast** spends the slot/pool unless `kind=item` (equipped grant: skip `SpendResource`, still apply combat statuses such as Armor of Agathys). At-will: no d6/dawn recharge.
- **Separate resource groups** for multiclass: `spell_slots` (Warlock excluded), `pact` (Warlock only; L4 = **2×2nd-level**, not 3rd), ki, sorcery points, Channel Divinity.
- Cast of combat spells writes a **status** (see below): Armor of Agathys, Mage Armor, Shield, Shield of Faith, Barkskin, Haste, Sanctuary.
- **Long rest is TODO** (refill pools, re-prepare).

## Skills

- Under ability scores: six **saves** in the ability boxes (score + mod, divider, save) and 18 **skills** in **PHB order**, 2 columns (3 on wide) — **not** grouped by ability.
- Bonus is derived: ability mod + PB (×2 expertise). Recalculates after ASI / level / PB. Equipped `skill_proficiency` overlays proficient while worn (not persisted; expertise stays owner-only). Equipped `ability_bonus` / `ability_penalty` apply to scores for mods / saves / AC / skills while worn. **Numeric item bonuses to skills/saves still TODO.**
- Auto-grant class saves and PHB background (and a few race) skills; owner can toggle proficient / expertise freely.
- Check / Save: same modal as spells (`1d20+bonus`). DM read-only.

## Inventory

- Island `#sheet-inventory`: Equipped / Pack / Consumables / **Add** wizard. Owner mutations; DM read-only.
- **Add wizard:** type → Weapon / Consumable / Armor / Jewelry. Weapons, armor, and jewelry: searchable PHB (or slot) bases → Common or Magic. Consumables keep search → preview → add.
- **Common +N:** weapons `atk_bonus`+`dmg_bonus`; armor/jewelry `ac_bonus` (Armor +1 / ring of protection style). Versatile 2H grip stays weapon-only.
- **Magic:** item-level fields plus a slim features list pre-filled from the base (versatile, thrown, martial/simple, stealth disadv, heavy, shield, …). Add via generic-feature catalog (including grant/skill/resistance pickers), magic item/spell search, or a pasted 5e14 URL. Vorpal unpacks into several one-stat rows. Unknown URL → one `note`.
- **Features** are one stat + value + origin (`base` / `common` / `magical` / `unique`). Pickers: `grant_spell` / `grant_cantrip` (value = catalog spell id, dropdown filtered by level), `skill_proficiency`, `resistance` / `vulnerability` / `immunity`, `condition_immunity`, `ability_bonus` / `ability_penalty`. Base hit lives on the weapon (`WeaponDTO.BaseHit`), not as a feature. `ArmorDTO` / `JewelryDTO` hold category/AC/slot. Do not encode Moonblade d100 tables — the player stores the chosen features. Catalog item id for builder weapons/armor is the **base**; custom name holds “Moonblade”.
- Slots: armor, shield, main_hand, off_hand, head, cloak, neck, ring_1/ring_2, gloves, boots, belt. Two-handed occupies main+off. Shield occupies shield **and** off-hand. Rings auto-pick. Equip **blocks** if occupied (names the other item); no auto-swap.
- **House rule:** attune on equip / unattune on unequip, max **3**. Equipped grants apply only while worn (virtual prepared spells, skill overlay, ability bonus, resistance line under defense). Unequip drops them immediately.
- Equipped weapon subtitle is character-relative (`+6 to hit · 1d8+4 slashing`): melee STR, ranged DEX, finesse = higher of STR/DEX (shown); thrown melee keeps that melee ability. To hit is ability + PB if proficient + item `atk_bonus`. Damage is weapon dice + ability + item `dmg_bonus`; extra-dice features are a second IRL/roll line. Proficiency uses encoded PHB 2014 class weapon lists (multiclass union; match `WeaponDTO.BaseSlug` + `InheritWeaponSlug`, else simple/martial). Versatile 2H uses `VersatileDice` unless throwing (stays 1H dice). Pack stays item-only (`BaseHit` + features). **Use** on equipped main/off-hand opens the spell-style modal (IRL + Auto-calculate; owner rolls, DM read-only; no ammo/charge spend, distinct from consumable `POST .../use`).
- Catalog item/spell/base search is Unicode case-insensitive (SQLite `LIKE`/`NOCASE` is ASCII-only). Debounce 300ms; do not remount the search input. Wizard 422 keeps `HX-Retarget` / innerHTML swap.
- Base weapons link to `https://5e14.dnd.su/articles/inventory/96-arms/`; armor/shields to `https://5e14.dnd.su/articles/inventory/95-armor-and-shields/` (encoded PHB tables; **no HTML scrape**). Magic artifacts keep `catalog_items` 5e14 item URLs. RU locale uses `desc_ru` if present; otherwise EN SRD + the 5e14 link.
- Stack mundane/consumables with no **manual** features; never stack magic/stub/configured rows.
- Consumable **Use** decrements (delete at 0). `potion-of-speed` → haste-like status `source=potion`. Healing potions roll into current HP. Unknown: decrement only.
- **TODO:** skill/save *numeric* item bonuses beyond proficiency, starting equipment, rest charge refill, d6/dawn recharge.

## Combat / statuses

- **Defense** (`sheet_combat.html`): current HP and **temp HP** ± open a **modal** (`#hp-adjust-modal`; do not step by 1 on the button). HP − is typed damage (PHB types): equipped item **immunity → 0**, **resistance → half (round down)**, **vulnerability → double**; resist+vuln cancel (×1). **RAW temp HP first** (`ReduceStackedTemp`), remainder hits current HP. HP + heals current HP only (clamp to HPMax, no temp restore). Temp ± is an amount only (no type, no resist; `AdjustTempHP` / `other-temp`). Owner or campaign DM (`RequireCombatEdit`); viewers without edit do not get ±. Death-save pips, derived **AC** / **speed**, short **resistance/immunity** line from equipped features. Death saves clear when current HP > 0. Apply result (e.g. “10 slashing, resistance, 5 applied”) shows in the modal; `#sheet-vitals` refreshes for owner and DM via **SSE**.
- **Statuses** (`sheet_statuses.html`): separate block — name, numeric formula, duration, source (`spell` / `potion` / `ability` / `other`), **Remove** (owner or campaign DM).
- **House rule:** temp HP from **different sources stack**. Recasting the **same** spell slug refreshes that row. Manual temp ± uses an `other-temp` status.
- AC/speed: unarmored (10+DEX, Monk UD, Barbarian UD, Monk unarmored movement) plus **worn armor/shield** and statuses. Light +DEX, medium +min(DEX, cap), heavy no DEX, shield +2. Monk UD off if armor or shield; Barb UD off if armor (shield OK). Mage Armor `ACBase` ignored while wearing armor. Instance AC/speed overlays stack. Duration is **displayed**, not ticked down. **Numeric equipment bonuses to skills/saves still TODO.**

## Data

- Migrations `001`–`014`. `data/` is not in git. Do not commit `.cursor/mcp.json`.
- Restart (or let Air restart) after adding SQL.

## Process

- Pause and ask Ivan on product forks.
- Don’t commit secrets. Don’t commit unless asked. Don’t push unless asked.
