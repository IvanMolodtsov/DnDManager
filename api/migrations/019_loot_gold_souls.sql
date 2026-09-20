-- Battle loot, per-character gold, and campaign souls.

ALTER TABLE characters ADD COLUMN gold INTEGER NOT NULL DEFAULT 0;

ALTER TABLE campaigns ADD COLUMN souls INTEGER NOT NULL DEFAULT 0;
ALTER TABLE campaigns ADD COLUMN souls_cap INTEGER NOT NULL DEFAULT 5000;

ALTER TABLE battles ADD COLUMN loot_generated INTEGER NOT NULL DEFAULT 0;
ALTER TABLE battles ADD COLUMN souls_applied INTEGER NOT NULL DEFAULT 0;

ALTER TABLE battle_units ADD COLUMN cr REAL NOT NULL DEFAULT 0;
ALTER TABLE battle_units ADD COLUMN cr_label TEXT NOT NULL DEFAULT '0';

UPDATE battle_units
SET cr = COALESCE((SELECT cr FROM catalog_monsters WHERE catalog_monsters.id = battle_units.catalog_monster_id), 0),
    cr_label = COALESCE((SELECT cr_label FROM catalog_monsters WHERE catalog_monsters.id = battle_units.catalog_monster_id), '0')
WHERE catalog_monster_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS battle_loot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    battle_id INTEGER NOT NULL REFERENCES battles(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    name_ru TEXT NOT NULL DEFAULT '',
    catalog_item_id INTEGER REFERENCES catalog_items(id),
    catalog_spell_id INTEGER REFERENCES catalog_spells(id),
    qty INTEGER NOT NULL DEFAULT 1,
    rarity TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_battle_loot_battle ON battle_loot(battle_id, sort_order, id);
