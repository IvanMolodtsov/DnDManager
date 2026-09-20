-- Campaign encounters: one battle row per campaign until the DM dismisses it.

CREATE TABLE IF NOT EXISTS battles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL UNIQUE REFERENCES campaigns(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    round INTEGER NOT NULL DEFAULT 1,
    active_index INTEGER NOT NULL DEFAULT 0,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS battle_units (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    battle_id INTEGER NOT NULL REFERENCES battles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    character_id INTEGER REFERENCES characters(id) ON DELETE CASCADE,
    catalog_monster_id INTEGER REFERENCES catalog_monsters(id),
    name TEXT NOT NULL,
    initiative INTEGER NOT NULL DEFAULT 0,
    hp_current INTEGER NOT NULL DEFAULT 0,
    hp_max INTEGER NOT NULL DEFAULT 0,
    temp_hp INTEGER NOT NULL DEFAULT 0,
    ac INTEGER NOT NULL DEFAULT 10,
    dex INTEGER NOT NULL DEFAULT 10,
    death_success INTEGER NOT NULL DEFAULT 0,
    death_fail INTEGER NOT NULL DEFAULT 0,
    dead INTEGER NOT NULL DEFAULT 0,
    escaped INTEGER NOT NULL DEFAULT 0,
    knocked INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    resist_json TEXT NOT NULL DEFAULT '{}',
    source_url_ru TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_battle_units_battle ON battle_units(battle_id, sort_order);
