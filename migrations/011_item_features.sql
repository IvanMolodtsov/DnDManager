-- Per-instance item features (mechanics) plus optional RU catalog body.
-- Overlay columns on character_items stay as a derived cache of feature sums.

ALTER TABLE catalog_items ADD COLUMN desc_ru TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS character_item_features (
    id INTEGER PRIMARY KEY,
    character_item_id INTEGER NOT NULL REFERENCES character_items(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'manual',
    name_en TEXT NOT NULL DEFAULT '',
    name_ru TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    catalog_kind TEXT NOT NULL DEFAULT '',
    catalog_id INTEGER NOT NULL DEFAULT 0,
    catalog_slug TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    atk_bonus INTEGER NOT NULL DEFAULT 0,
    dmg_bonus INTEGER NOT NULL DEFAULT 0,
    extra_damage TEXT NOT NULL DEFAULT '',
    damage_dice TEXT NOT NULL DEFAULT '',
    damage_type TEXT NOT NULL DEFAULT '',
    crit_range INTEGER NOT NULL DEFAULT 0,
    ac_bonus INTEGER NOT NULL DEFAULT 0,
    ac_base INTEGER NOT NULL DEFAULT 0,
    ac_floor INTEGER NOT NULL DEFAULT 0,
    speed_bonus INTEGER NOT NULL DEFAULT 0,
    speed_mult INTEGER NOT NULL DEFAULT 0,
    int_score INTEGER NOT NULL DEFAULT 0,
    wis_score INTEGER NOT NULL DEFAULT 0,
    cha_score INTEGER NOT NULL DEFAULT 0,
    property_slug TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_item_features_item ON character_item_features(character_item_id);

INSERT INTO character_item_features (
    character_item_id, sort_order, source, name_en, name_ru, notes,
    extra_damage, atk_bonus, dmg_bonus, ac_bonus, ac_base, ac_floor, speed_bonus, speed_mult
)
SELECT id, 0, 'manual', 'Configured bonuses', 'Настроенные бонусы', '',
       extra_damage, atk_bonus, dmg_bonus, ac_bonus, ac_base, ac_floor, speed_bonus, speed_mult
FROM character_items
WHERE atk_bonus != 0 OR dmg_bonus != 0 OR extra_damage != ''
   OR ac_bonus != 0 OR ac_base != 0 OR ac_floor != 0
   OR speed_bonus != 0 OR speed_mult != 0;
