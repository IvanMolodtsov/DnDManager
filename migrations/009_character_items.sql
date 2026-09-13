-- Catalog item mechanics + per-character inventory instances.
-- Generated seed rows live in 010 (scripts/genitems).

ALTER TABLE catalog_items ADD COLUMN desc_en TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_items ADD COLUMN armor_category TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_items ADD COLUMN ac_base INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN dex_max INTEGER NOT NULL DEFAULT -1;
ALTER TABLE catalog_items ADD COLUMN stealth_disadv INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN str_min INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN weapon_category TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_items ADD COLUMN versatile_dice TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_items ADD COLUMN range_normal INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN range_long INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN suggested_slot TEXT NOT NULL DEFAULT '';
ALTER TABLE catalog_items ADD COLUMN requires_attunement INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN consumable INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN charges_max INTEGER NOT NULL DEFAULT 0;
ALTER TABLE catalog_items ADD COLUMN is_stub INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS character_items (
    id INTEGER PRIMARY KEY,
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    catalog_item_id INTEGER NOT NULL REFERENCES catalog_items(id),
    quantity INTEGER NOT NULL DEFAULT 1,
    equipped_slot TEXT NOT NULL DEFAULT '',
    attuned INTEGER NOT NULL DEFAULT 0,
    charges_current INTEGER NOT NULL DEFAULT 0,
    charges_max INTEGER NOT NULL DEFAULT 0,
    custom_name TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    atk_bonus INTEGER NOT NULL DEFAULT 0,
    dmg_bonus INTEGER NOT NULL DEFAULT 0,
    extra_damage TEXT NOT NULL DEFAULT '',
    ac_bonus INTEGER NOT NULL DEFAULT 0,
    ac_base INTEGER NOT NULL DEFAULT 0,
    ac_floor INTEGER NOT NULL DEFAULT 0,
    speed_bonus INTEGER NOT NULL DEFAULT 0,
    speed_mult INTEGER NOT NULL DEFAULT 0,
    two_handed INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_character_items_char ON character_items(character_id);
CREATE INDEX IF NOT EXISTS idx_catalog_items_slug ON catalog_items(slug);
CREATE INDEX IF NOT EXISTS idx_catalog_items_name ON catalog_items(name_en);
