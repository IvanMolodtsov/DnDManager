-- Permanent mutations and extra feature rows (one mutation per body part).

CREATE TABLE IF NOT EXISTS character_mutations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    body_part TEXT NOT NULL,
    monster_id INTEGER REFERENCES catalog_monsters(id),
    name_en TEXT NOT NULL DEFAULT '',
    name_ru TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT '',
    cr REAL NOT NULL DEFAULT 0,
    cr_label TEXT NOT NULL DEFAULT '0',
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT '',
    UNIQUE (character_id, body_part)
);

CREATE INDEX IF NOT EXISTS idx_character_mutations_char ON character_mutations(character_id);

CREATE TABLE IF NOT EXISTS character_mutation_features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mutation_id INTEGER NOT NULL REFERENCES character_mutations(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    stat TEXT NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    extra TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_character_mutation_features_mut ON character_mutation_features(mutation_id, sort_order, id);
