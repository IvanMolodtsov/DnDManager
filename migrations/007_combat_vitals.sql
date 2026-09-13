-- Combat vitals: temp HP, death saves, and dismissible spell effects.
-- Armor/speed from inventory is still TODO.

ALTER TABLE characters ADD COLUMN hp_temp INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN death_success INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN death_fail INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS character_effects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    kind TEXT NOT NULL,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    source_spell_id INTEGER,
    ac_bonus INTEGER NOT NULL DEFAULT 0,
    ac_base INTEGER NOT NULL DEFAULT 0,
    ac_floor INTEGER NOT NULL DEFAULT 0,
    speed_bonus INTEGER NOT NULL DEFAULT 0,
    speed_mult INTEGER NOT NULL DEFAULT 0,
    temp_hp INTEGER NOT NULL DEFAULT 0,
    tags TEXT NOT NULL DEFAULT '',
    UNIQUE(character_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_character_effects_char ON character_effects(character_id);
