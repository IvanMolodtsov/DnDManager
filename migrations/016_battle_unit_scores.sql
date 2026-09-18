-- Additive battle unit snapshot: six ability scores (dex already exists).
-- Resist/immune/vuln (and optional save flags) stay in resist_json.
-- Do not wipe live fights or catalog rows.

ALTER TABLE battle_units ADD COLUMN str INTEGER NOT NULL DEFAULT 10;
ALTER TABLE battle_units ADD COLUMN con INTEGER NOT NULL DEFAULT 10;
ALTER TABLE battle_units ADD COLUMN intel INTEGER NOT NULL DEFAULT 10;
ALTER TABLE battle_units ADD COLUMN wis INTEGER NOT NULL DEFAULT 10;
ALTER TABLE battle_units ADD COLUMN cha INTEGER NOT NULL DEFAULT 10;
