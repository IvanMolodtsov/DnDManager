CREATE TABLE IF NOT EXISTS character_skills (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    skill_slug TEXT NOT NULL,
    proficient INTEGER NOT NULL DEFAULT 0,
    expertise INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, skill_slug)
);

CREATE TABLE IF NOT EXISTS character_saves (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    ability TEXT NOT NULL,
    proficient INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, ability)
);
