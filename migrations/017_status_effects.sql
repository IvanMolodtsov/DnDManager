-- DM-applied statuses: extra columns on character_effects, monster rows, PHB conditions.
-- Old effect rows stay working: not hidden, duration 0 (not turn-limited), remove_on_battle_end 0.

ALTER TABLE character_effects ADD COLUMN hidden INTEGER NOT NULL DEFAULT 0;
ALTER TABLE character_effects ADD COLUMN remove_on_battle_end INTEGER NOT NULL DEFAULT 0;
ALTER TABLE character_effects ADD COLUMN duration_turns INTEGER NOT NULL DEFAULT 0;
ALTER TABLE character_effects ADD COLUMN damage_formula TEXT NOT NULL DEFAULT '';
ALTER TABLE character_effects ADD COLUMN damage_type TEXT NOT NULL DEFAULT '';
ALTER TABLE character_effects ADD COLUMN source_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS battle_unit_effects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    unit_id INTEGER NOT NULL REFERENCES battle_units(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'condition',
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'other',
    duration_key TEXT NOT NULL DEFAULT '',
    formula_en TEXT NOT NULL DEFAULT '',
    formula_ru TEXT NOT NULL DEFAULT '',
    ac_bonus INTEGER NOT NULL DEFAULT 0,
    ac_base INTEGER NOT NULL DEFAULT 0,
    ac_floor INTEGER NOT NULL DEFAULT 0,
    speed_bonus INTEGER NOT NULL DEFAULT 0,
    speed_mult INTEGER NOT NULL DEFAULT 0,
    temp_hp INTEGER NOT NULL DEFAULT 0,
    tags TEXT NOT NULL DEFAULT '',
    hidden INTEGER NOT NULL DEFAULT 0,
    remove_on_battle_end INTEGER NOT NULL DEFAULT 1,
    duration_turns INTEGER NOT NULL DEFAULT 0,
    damage_formula TEXT NOT NULL DEFAULT '',
    damage_type TEXT NOT NULL DEFAULT '',
    source_url TEXT NOT NULL DEFAULT '',
    UNIQUE(unit_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_battle_unit_effects_unit ON battle_unit_effects(unit_id);

-- PHB 2014 Appendix A conditions. 5e14 link is the conditions article
-- https://5e14.dnd.su/articles/mechanics/27-conditions/ (same idea as the PHB arms table).
-- EN provenance is 5eapi /api/2014/conditions/{slug}. Do not invent /condition/poisoned/.
-- burning is a common combat overlay (not a PHB condition); no 5e14 path.

CREATE TABLE IF NOT EXISTS catalog_conditions (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT '',
    damage_formula TEXT NOT NULL DEFAULT '',
    damage_type TEXT NOT NULL DEFAULT '',
    is_phb INTEGER NOT NULL DEFAULT 1
);

INSERT INTO catalog_conditions (id, slug, name_en, name_ru, source_url, source_url_ru, damage_formula, damage_type, is_phb) VALUES
(1, 'blinded', 'Blinded', 'Ослеплённый', 'https://www.dnd5eapi.co/api/2014/conditions/blinded', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(2, 'charmed', 'Charmed', 'Очарованный', 'https://www.dnd5eapi.co/api/2014/conditions/charmed', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(3, 'deafened', 'Deafened', 'Оглохший', 'https://www.dnd5eapi.co/api/2014/conditions/deafened', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(4, 'exhaustion', 'Exhaustion', 'Истощённый', 'https://www.dnd5eapi.co/api/2014/conditions/exhaustion', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(5, 'frightened', 'Frightened', 'Испуганный', 'https://www.dnd5eapi.co/api/2014/conditions/frightened', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(6, 'grappled', 'Grappled', 'Схваченный', 'https://www.dnd5eapi.co/api/2014/conditions/grappled', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(7, 'incapacitated', 'Incapacitated', 'Недееспособный', 'https://www.dnd5eapi.co/api/2014/conditions/incapacitated', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(8, 'invisible', 'Invisible', 'Невидимый', 'https://www.dnd5eapi.co/api/2014/conditions/invisible', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(9, 'paralyzed', 'Paralyzed', 'Парализованный', 'https://www.dnd5eapi.co/api/2014/conditions/paralyzed', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(10, 'petrified', 'Petrified', 'Окаменевший', 'https://www.dnd5eapi.co/api/2014/conditions/petrified', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(11, 'poisoned', 'Poisoned', 'Отравленный', 'https://www.dnd5eapi.co/api/2014/conditions/poisoned', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(12, 'prone', 'Prone', 'Сбитый с ног', 'https://www.dnd5eapi.co/api/2014/conditions/prone', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(13, 'restrained', 'Restrained', 'Опутанный', 'https://www.dnd5eapi.co/api/2014/conditions/restrained', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(14, 'stunned', 'Stunned', 'Ошеломлённый', 'https://www.dnd5eapi.co/api/2014/conditions/stunned', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(15, 'unconscious', 'Unconscious', 'Бессознательный', 'https://www.dnd5eapi.co/api/2014/conditions/unconscious', 'https://5e14.dnd.su/articles/mechanics/27-conditions/', '', '', 1),
(16, 'burning', 'Burning', 'Горение', '', '', '1d6', 'fire', 0);
