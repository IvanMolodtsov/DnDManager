-- Persistent PC companions (Beast Master, familiar, item/weapon). Not characters.
-- Battle units gain companion_id so they sort immediately after the master.

CREATE TABLE IF NOT EXISTS character_companions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    catalog_monster_id INTEGER REFERENCES catalog_monsters(id),
    item_id INTEGER REFERENCES character_items(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    name_en TEXT NOT NULL DEFAULT '',
    name_ru TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    armor_class INTEGER NOT NULL DEFAULT 10,
    hp_current INTEGER NOT NULL DEFAULT 0,
    hp_max INTEGER NOT NULL DEFAULT 0,
    str INTEGER NOT NULL DEFAULT 10,
    dex INTEGER NOT NULL DEFAULT 10,
    con INTEGER NOT NULL DEFAULT 10,
    intel INTEGER NOT NULL DEFAULT 10,
    wis INTEGER NOT NULL DEFAULT 10,
    cha INTEGER NOT NULL DEFAULT 10,
    resist_json TEXT NOT NULL DEFAULT '{}',
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT '',
    can_attack INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_character_companions_char ON character_companions(character_id, sort_order, id);

ALTER TABLE battle_units ADD COLUMN companion_id INTEGER REFERENCES character_companions(id) ON DELETE SET NULL;

-- PHB 2014 Ranger archetype missing from the original subclass seed.
-- 5eapi subclass URL + parent 5e14 ranger page (same pattern as Hunter).
INSERT OR IGNORE INTO catalog_subclasses (id, class_id, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(16, 9, 'beast-master', 'Beast Master', 'Повелитель зверей',
 'https://www.dnd5eapi.co/api/2014/subclasses/beast-master', 'https://5e14.dnd.su/class/97-ranger/');

INSERT OR IGNORE INTO catalog_features (id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(700, 'subclass', 16, 3, 'rangers-companion', 'Ranger''s Companion', 'Спутник следопыта',
 'https://www.dnd5eapi.co/api/2014/features/rangers-companion', 'https://5e14.dnd.su/class/97-ranger/'),
(701, 'class', 12, 0, 'pact-of-the-chain', 'Pact of the Chain', 'Договор цепи',
 'https://www.dnd5eapi.co/api/2014/features/pact-of-the-chain', 'https://5e14.dnd.su/class/104-warlock/');

-- Equipped-item hook when no named "golem stones" catalog row exists.
INSERT OR IGNORE INTO catalog_stat_features (id, slug, name_en, name_ru, stat, default_value, origin, source_url, sort_order) VALUES
(70, 'grant-companion', 'Grant a companion', 'Даёт спутника', 'grant_companion', '', 'magical', '', 365);
