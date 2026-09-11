-- Thin 5e catalog (examples only) + character progression fields.

CREATE TABLE IF NOT EXISTS catalog_races (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT '',
    ability_bonuses TEXT NOT NULL DEFAULT '{}',
    speed INTEGER NOT NULL DEFAULT 30,
    hp_bonus_per_level INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS catalog_backgrounds (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT '',
    ability_bonuses TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS catalog_classes (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    hit_die INTEGER NOT NULL,
    subclass_level INTEGER NOT NULL,
    asi_levels TEXT NOT NULL DEFAULT '[4,8,12,16,19]',
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT '',
    ability_bonuses TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS catalog_subclasses (
    id INTEGER PRIMARY KEY,
    class_id INTEGER NOT NULL REFERENCES catalog_classes(id),
    slug TEXT NOT NULL UNIQUE,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS catalog_features (
    id INTEGER PRIMARY KEY,
    source_kind TEXT NOT NULL,
    source_id INTEGER NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
    slug TEXT NOT NULL,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    source_url_ru TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_catalog_features_source ON catalog_features(source_kind, source_id, level);
CREATE INDEX IF NOT EXISTS idx_catalog_subclasses_class ON catalog_subclasses(class_id);

CREATE TABLE IF NOT EXISTS character_drafts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    step INTEGER NOT NULL DEFAULT 1,
    name TEXT NOT NULL DEFAULT '',
    race_id INTEGER REFERENCES catalog_races(id),
    background_id INTEGER REFERENCES catalog_backgrounds(id),
    class_id INTEGER REFERENCES catalog_classes(id),
    subclass_id INTEGER REFERENCES catalog_subclasses(id),
    abilities_json TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(owner_id, campaign_id)
);

CREATE TABLE IF NOT EXISTS character_class_levels (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    class_id INTEGER NOT NULL REFERENCES catalog_classes(id),
    subclass_id INTEGER REFERENCES catalog_subclasses(id),
    levels INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (character_id, class_id)
);

CREATE TABLE IF NOT EXISTS character_features (
    character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    feature_id INTEGER NOT NULL REFERENCES catalog_features(id),
    PRIMARY KEY (character_id, feature_id)
);

ALTER TABLE characters ADD COLUMN race_id INTEGER REFERENCES catalog_races(id);
ALTER TABLE characters ADD COLUMN background_id INTEGER REFERENCES catalog_backgrounds(id);
ALTER TABLE characters ADD COLUMN hp_max INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN hp_current INTEGER NOT NULL DEFAULT 0;
ALTER TABLE characters ADD COLUMN proficiency_bonus INTEGER NOT NULL DEFAULT 2;

-- Races
INSERT INTO catalog_races (id, slug, name_en, name_ru, source_url, source_url_ru, ability_bonuses, speed, hp_bonus_per_level) VALUES
(1, 'human', 'Human', 'Человек',
 'https://www.dnd5eapi.co/api/2014/races/human', 'https://5e14.dnd.su/race/81-human/',
 '{"str":1,"dex":1,"con":1,"int":1,"wis":1,"cha":1}', 30, 0),
(2, 'hill-dwarf', 'Hill Dwarf', 'Холмовой дварф',
 'https://www.dnd5eapi.co/api/2014/subraces/hill-dwarf', 'https://5e14.dnd.su/race/78-dwarf/',
 '{"con":2,"wis":1}', 25, 1),
(3, 'high-elf', 'High Elf', 'Высший эльф',
 'https://www.dnd5eapi.co/api/2014/subraces/high-elf', 'https://5e14.dnd.su/race/79-elf/',
 '{"dex":2,"int":1}', 30, 0);

-- Backgrounds (2014: no ability bonuses)
INSERT INTO catalog_backgrounds (id, slug, name_en, name_ru, source_url, source_url_ru, ability_bonuses) VALUES
(1, 'acolyte', 'Acolyte', 'Прислужник',
 'https://www.dnd5eapi.co/api/2014/backgrounds/acolyte', 'https://5e14.dnd.su/backgrounds/766-acolyte/', '{}'),
(2, 'soldier', 'Soldier', 'Солдат',
 '', 'https://5e14.dnd.su/backgrounds/767-soldier/', '{}'),
(3, 'sage', 'Sage', 'Мудрец',
 '', 'https://5e14.dnd.su/backgrounds/762-sage/', '{}');

-- Classes: Fighter subclass 3 (extra ASI 6/14), Wizard subclass 2, Cleric subclass 1
INSERT INTO catalog_classes (id, slug, name_en, name_ru, hit_die, subclass_level, asi_levels, source_url, source_url_ru, ability_bonuses) VALUES
(1, 'fighter', 'Fighter', 'Воин', 10, 3, '[4,6,8,12,14,16,19]',
 'https://www.dnd5eapi.co/api/2014/classes/fighter', 'https://5e14.dnd.su/class/91-fighter/', '{}'),
(2, 'wizard', 'Wizard', 'Волшебник', 6, 2, '[4,8,12,16,19]',
 'https://www.dnd5eapi.co/api/2014/classes/wizard', 'https://5e14.dnd.su/class/105-wizard/', '{}'),
(3, 'cleric', 'Cleric', 'Жрец', 8, 1, '[4,8,12,16,19]',
 'https://www.dnd5eapi.co/api/2014/classes/cleric', 'https://5e14.dnd.su/class/89-cleric/', '{}');

INSERT INTO catalog_subclasses (id, class_id, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(1, 1, 'champion', 'Champion', 'Чемпион',
 'https://www.dnd5eapi.co/api/2014/subclasses/champion', 'https://5e14.dnd.su/class/91-fighter/'),
(2, 1, 'battle-master', 'Battle Master', 'Мастер боевых искусств',
 'https://www.dnd5eapi.co/api/2014/subclasses/battle-master', 'https://5e14.dnd.su/class/91-fighter/'),
(3, 2, 'evocation', 'School of Evocation', 'Школа воплощения',
 'https://www.dnd5eapi.co/api/2014/subclasses/evocation', 'https://5e14.dnd.su/class/105-wizard/'),
(4, 2, 'abjuration', 'School of Abjuration', 'Школа ограждения',
 'https://www.dnd5eapi.co/api/2014/subclasses/abjuration', 'https://5e14.dnd.su/class/105-wizard/'),
(5, 3, 'life', 'Life Domain', 'Домен жизни',
 'https://www.dnd5eapi.co/api/2014/subclasses/life', 'https://5e14.dnd.su/class/89-cleric/'),
(6, 3, 'light', 'Light Domain', 'Домен света',
 'https://www.dnd5eapi.co/api/2014/subclasses/light', 'https://5e14.dnd.su/class/89-cleric/');

-- Race features
INSERT INTO catalog_features (id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(1, 'race', 1, 1, 'extra-language', 'Extra Language', 'Дополнительный язык',
 'https://www.dnd5eapi.co/api/2014/traits/extra-language', 'https://5e14.dnd.su/race/81-human/'),
(2, 'race', 2, 1, 'dwarven-resilience', 'Dwarven Resilience', 'Дварфийская устойчивость',
 'https://www.dnd5eapi.co/api/2014/traits/dwarven-resilience', 'https://5e14.dnd.su/race/78-dwarf/'),
(3, 'race', 2, 1, 'dwarven-toughness', 'Dwarven Toughness', 'Дварфийская выносливость',
 'https://www.dnd5eapi.co/api/2014/traits/dwarven-toughness', 'https://5e14.dnd.su/race/78-dwarf/'),
(4, 'race', 3, 1, 'darkvision', 'Darkvision', 'Тёмное зрение',
 'https://www.dnd5eapi.co/api/2014/traits/darkvision', 'https://5e14.dnd.su/race/79-elf/'),
(5, 'race', 3, 1, 'elf-cantrip', 'Cantrip', 'Заговор',
 'https://www.dnd5eapi.co/api/2014/traits/elf-cantrip', 'https://5e14.dnd.su/race/79-elf/');

-- Background features
INSERT INTO catalog_features (id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(10, 'background', 1, 1, 'shelter-of-the-faithful', 'Shelter of the Faithful', 'Укрытие верных',
 'https://www.dnd5eapi.co/api/2014/backgrounds/acolyte', 'https://5e14.dnd.su/backgrounds/766-acolyte/'),
(11, 'background', 2, 1, 'military-rank', 'Military Rank', 'Воинское звание',
 '', 'https://5e14.dnd.su/backgrounds/767-soldier/'),
(12, 'background', 3, 1, 'researcher', 'Researcher', 'Исследователь',
 '', 'https://5e14.dnd.su/backgrounds/762-sage/');

-- Fighter features
INSERT INTO catalog_features (id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(20, 'class', 1, 1, 'fighting-style', 'Fighting Style', 'Боевой стиль',
 'https://www.dnd5eapi.co/api/2014/features/fighter-fighting-style', 'https://5e14.dnd.su/class/91-fighter/'),
(21, 'class', 1, 1, 'second-wind', 'Second Wind', 'Второе дыхание',
 'https://www.dnd5eapi.co/api/2014/features/second-wind', 'https://5e14.dnd.su/class/91-fighter/'),
(22, 'class', 1, 2, 'action-surge', 'Action Surge', 'Всплеск действий',
 'https://www.dnd5eapi.co/api/2014/features/action-surge-1-use', 'https://5e14.dnd.su/class/91-fighter/'),
(23, 'class', 1, 3, 'martial-archetype', 'Martial Archetype', 'Воинский архетип',
 'https://www.dnd5eapi.co/api/2014/features/martial-archetype', 'https://5e14.dnd.su/class/91-fighter/'),
(24, 'class', 1, 4, 'fighter-asi-4', 'Ability Score Improvement', 'Увеличение характеристик',
 'https://www.dnd5eapi.co/api/2014/features/fighter-ability-score-improvement-1', 'https://5e14.dnd.su/class/91-fighter/');

-- Wizard features
INSERT INTO catalog_features (id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(30, 'class', 2, 1, 'wizard-spellcasting', 'Spellcasting', 'Использование заклинаний',
 'https://www.dnd5eapi.co/api/2014/features/wizard-spellcasting', 'https://5e14.dnd.su/class/105-wizard/'),
(31, 'class', 2, 1, 'arcane-recovery', 'Arcane Recovery', 'Магическое восстановление',
 'https://www.dnd5eapi.co/api/2014/features/arcane-recovery', 'https://5e14.dnd.su/class/105-wizard/'),
(32, 'class', 2, 2, 'arcane-tradition', 'Arcane Tradition', 'Магическая традиция',
 'https://www.dnd5eapi.co/api/2014/features/arcane-tradition', 'https://5e14.dnd.su/class/105-wizard/'),
(33, 'class', 2, 4, 'wizard-asi-4', 'Ability Score Improvement', 'Увеличение характеристик',
 'https://www.dnd5eapi.co/api/2014/features/wizard-ability-score-improvement-1', 'https://5e14.dnd.su/class/105-wizard/');

-- Cleric features
INSERT INTO catalog_features (id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(40, 'class', 3, 1, 'cleric-spellcasting', 'Spellcasting', 'Использование заклинаний',
 'https://www.dnd5eapi.co/api/2014/features/cleric-spellcasting', 'https://5e14.dnd.su/class/89-cleric/'),
(41, 'class', 3, 1, 'divine-domain', 'Divine Domain', 'Божественный домен',
 'https://www.dnd5eapi.co/api/2014/features/divine-domain', 'https://5e14.dnd.su/class/89-cleric/'),
(42, 'class', 3, 2, 'channel-divinity', 'Channel Divinity', 'Божественный канал',
 'https://www.dnd5eapi.co/api/2014/features/channel-divinity-1-rest', 'https://5e14.dnd.su/class/89-cleric/'),
(43, 'class', 3, 4, 'cleric-asi-4', 'Ability Score Improvement', 'Увеличение характеристик',
 'https://www.dnd5eapi.co/api/2014/features/cleric-ability-score-improvement-1', 'https://5e14.dnd.su/class/89-cleric/');

-- Subclass features (RAW levels)
INSERT INTO catalog_features (id, source_kind, source_id, level, slug, name_en, name_ru, source_url, source_url_ru) VALUES
(50, 'subclass', 1, 3, 'improved-critical', 'Improved Critical', 'Улучшенный критический удар',
 'https://www.dnd5eapi.co/api/2014/features/improved-critical', 'https://5e14.dnd.su/class/91-fighter/'),
(51, 'subclass', 2, 3, 'combat-superiority', 'Combat Superiority', 'Боевое превосходство',
 'https://www.dnd5eapi.co/api/2014/features/combat-superiority', 'https://5e14.dnd.su/class/91-fighter/'),
(52, 'subclass', 3, 2, 'evocation-savant', 'Evocation Savant', 'Знаток воплощения',
 'https://www.dnd5eapi.co/api/2014/features/evocation-savant', 'https://5e14.dnd.su/class/105-wizard/'),
(53, 'subclass', 3, 2, 'sculpt-spells', 'Sculpt Spells', 'Лепка заклинаний',
 'https://www.dnd5eapi.co/api/2014/features/sculpt-spells', 'https://5e14.dnd.su/class/105-wizard/'),
(54, 'subclass', 4, 2, 'abjuration-savant', 'Abjuration Savant', 'Знаток ограждения',
 'https://www.dnd5eapi.co/api/2014/features/abjuration-savant', 'https://5e14.dnd.su/class/105-wizard/'),
(55, 'subclass', 4, 2, 'arcane-ward', 'Arcane Ward', 'Магическая защита',
 'https://www.dnd5eapi.co/api/2014/features/arcane-ward', 'https://5e14.dnd.su/class/105-wizard/'),
(56, 'subclass', 5, 1, 'disciple-of-life', 'Disciple of Life', 'Ученик жизни',
 'https://www.dnd5eapi.co/api/2014/features/disciple-of-life', 'https://5e14.dnd.su/class/89-cleric/'),
(57, 'subclass', 6, 1, 'warding-flare', 'Warding Flare', 'Оберегающая вспышка',
 'https://www.dnd5eapi.co/api/2014/features/warding-flare', 'https://5e14.dnd.su/class/89-cleric/');
