-- Weapon builder: PHB base flag, generic one-stat catalog, slim instance features.
-- catalog_features already holds class/race grants, so generic item stats live in catalog_stat_features.
-- Overlay int columns on character_items stay a derived cache of feature sums.

ALTER TABLE catalog_items ADD COLUMN is_base INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS catalog_stat_features (
    id INTEGER PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name_en TEXT NOT NULL,
    name_ru TEXT NOT NULL,
    stat TEXT NOT NULL,
    default_value TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT 'common',
    source_url TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

INSERT INTO catalog_stat_features (id, slug, name_en, name_ru, stat, default_value, origin, source_url, sort_order) VALUES
(1, 'atk-bonus', 'Attack bonus', 'Бонус атаки', 'atk_bonus', '1', 'common', '', 10),
(2, 'dmg-bonus', 'Damage bonus', 'Бонус урона', 'dmg_bonus', '1', 'common', '', 20),
(3, 'extra-dice', 'Extra dice', 'Доп. кости', 'extra_dice', '1d6', 'magical', '', 30),
(4, 'crit-range', 'Improved critical', 'Улучшенный крит', 'crit_range', '19', 'unique', '', 40),
(5, 'personality', 'Personality', 'Личность', 'personality', '12, 10, 12', 'unique', '', 50),
(6, 'note', 'Note', 'Заметка', 'note', '', 'unique', '', 60),
(7, 'ac-bonus', 'AC bonus', 'Бонус КД', 'ac_bonus', '1', 'magical', '', 70),
(8, 'simple', 'Simple', 'Простое', 'property', 'simple', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 100),
(9, 'martial', 'Martial', 'Воинское', 'property', 'martial', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 110),
(10, 'versatile', 'Versatile', 'Универсальное', 'versatile', '1d10', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 120),
(11, 'thrown', 'Thrown', 'Метательное', 'thrown', '20/60', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 130),
(12, 'finesse', 'Finesse', 'Фехтовальное', 'property', 'finesse', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 140),
(13, 'light', 'Light', 'Лёгкое', 'property', 'light', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 150),
(14, 'heavy', 'Heavy', 'Тяжёлое', 'property', 'heavy', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 160),
(15, 'reach', 'Reach', 'Досягаемость', 'property', 'reach', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 170),
(16, 'two-handed', 'Two-handed', 'Двуручное', 'property', 'two-handed', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 180),
(17, 'ammunition', 'Ammunition', 'Боеприпасы', 'property', 'ammunition', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 190),
(18, 'loading', 'Loading', 'Перезарядка', 'property', 'loading', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 200),
(19, 'special', 'Special', 'Особое', 'property', 'special', 'base', 'https://5e14.dnd.su/articles/inventory/96-arms/', 210);

UPDATE catalog_items SET is_base = 1, source_url_ru = 'https://5e14.dnd.su/articles/inventory/96-arms/'
WHERE slug IN (
    'club','dagger','greatclub','handaxe','javelin','light-hammer','mace','quarterstaff','sickle','spear',
    'crossbow-light','dart','shortbow','sling',
    'battleaxe','flail','glaive','greataxe','greatsword','halberd','lance','longsword','maul','morningstar',
    'pike','rapier','scimitar','shortsword','trident','war-pick','warhammer','whip',
    'blowgun','crossbow-hand','crossbow-heavy','longbow','net'
);

UPDATE catalog_items SET name_ru = 'Дубинка', properties = '["light"]', range_normal = 0, range_long = 0 WHERE slug = 'club';
UPDATE catalog_items SET name_ru = 'Кинжал', properties = '["finesse","light","thrown"]', range_normal = 20, range_long = 60 WHERE slug = 'dagger';
UPDATE catalog_items SET name_ru = 'Палица', properties = '["two-handed"]', range_normal = 0, range_long = 0 WHERE slug = 'greatclub';
UPDATE catalog_items SET name_ru = 'Ручной топор', properties = '["light","thrown"]', range_normal = 20, range_long = 60 WHERE slug = 'handaxe';
UPDATE catalog_items SET name_ru = 'Метательное копьё', properties = '["thrown"]', range_normal = 30, range_long = 120 WHERE slug = 'javelin';
UPDATE catalog_items SET name_ru = 'Лёгкий молот', properties = '["light","thrown"]', range_normal = 20, range_long = 60 WHERE slug = 'light-hammer';
UPDATE catalog_items SET name_ru = 'Булава', properties = '[]', range_normal = 0, range_long = 0 WHERE slug = 'mace';
UPDATE catalog_items SET name_ru = 'Боевой посох', properties = '["versatile"]', versatile_dice = '1d8', range_normal = 0, range_long = 0 WHERE slug = 'quarterstaff';
UPDATE catalog_items SET name_ru = 'Серп', properties = '["light"]', range_normal = 0, range_long = 0 WHERE slug = 'sickle';
UPDATE catalog_items SET name_ru = 'Копьё', properties = '["thrown","versatile"]', versatile_dice = '1d8', range_normal = 20, range_long = 60 WHERE slug = 'spear';
UPDATE catalog_items SET name_ru = 'Лёгкий арбалет', properties = '["ammunition","loading","two-handed"]' WHERE slug = 'crossbow-light';
UPDATE catalog_items SET name_ru = 'Дротик', properties = '["finesse","thrown"]', range_normal = 20, range_long = 60 WHERE slug = 'dart';
UPDATE catalog_items SET name_ru = 'Короткий лук', properties = '["ammunition","two-handed"]' WHERE slug = 'shortbow';
UPDATE catalog_items SET name_ru = 'Праща', properties = '["ammunition"]' WHERE slug = 'sling';
UPDATE catalog_items SET name_ru = 'Боевой топор', properties = '["versatile"]', versatile_dice = '1d10' WHERE slug = 'battleaxe';
UPDATE catalog_items SET name_ru = 'Цеп', properties = '[]' WHERE slug = 'flail';
UPDATE catalog_items SET name_ru = 'Глефа', properties = '["heavy","reach","two-handed"]' WHERE slug = 'glaive';
UPDATE catalog_items SET name_ru = 'Секира', properties = '["heavy","two-handed"]' WHERE slug = 'greataxe';
UPDATE catalog_items SET name_ru = 'Двуручный меч', properties = '["heavy","two-handed"]' WHERE slug = 'greatsword';
UPDATE catalog_items SET name_ru = 'Алебарда', properties = '["heavy","reach","two-handed"]' WHERE slug = 'halberd';
UPDATE catalog_items SET name_ru = 'Длинное копьё', properties = '["reach","special"]' WHERE slug = 'lance';
UPDATE catalog_items SET name_ru = 'Длинный меч', properties = '["versatile"]', versatile_dice = '1d10' WHERE slug = 'longsword';
UPDATE catalog_items SET name_ru = 'Молот', properties = '["heavy","two-handed"]' WHERE slug = 'maul';
UPDATE catalog_items SET name_ru = 'Моргенштерн', properties = '[]' WHERE slug = 'morningstar';
UPDATE catalog_items SET name_ru = 'Пика', properties = '["heavy","reach","two-handed"]' WHERE slug = 'pike';
UPDATE catalog_items SET name_ru = 'Рапира', properties = '["finesse"]' WHERE slug = 'rapier';
UPDATE catalog_items SET name_ru = 'Скимитар', properties = '["finesse","light"]' WHERE slug = 'scimitar';
UPDATE catalog_items SET name_ru = 'Короткий меч', properties = '["finesse","light"]' WHERE slug = 'shortsword';
UPDATE catalog_items SET name_ru = 'Трезубец', properties = '["thrown","versatile"]', versatile_dice = '1d8', range_normal = 20, range_long = 60 WHERE slug = 'trident';
UPDATE catalog_items SET name_ru = 'Боевая кирка', properties = '[]' WHERE slug = 'war-pick';
UPDATE catalog_items SET name_ru = 'Боевой молот', properties = '["versatile"]', versatile_dice = '1d10' WHERE slug = 'warhammer';
UPDATE catalog_items SET name_ru = 'Кнут', properties = '["finesse","reach"]' WHERE slug = 'whip';
UPDATE catalog_items SET name_ru = 'Духовая трубка', properties = '["ammunition","loading"]' WHERE slug = 'blowgun';
UPDATE catalog_items SET name_ru = 'Ручной арбалет', properties = '["ammunition","light","loading"]' WHERE slug = 'crossbow-hand';
UPDATE catalog_items SET name_ru = 'Тяжёлый арбалет', properties = '["ammunition","heavy","loading","two-handed"]' WHERE slug = 'crossbow-heavy';
UPDATE catalog_items SET name_ru = 'Длинный лук', properties = '["ammunition","heavy","two-handed"]' WHERE slug = 'longbow';
UPDATE catalog_items SET name_ru = 'Сеть', properties = '["thrown","special"]', range_normal = 5, range_long = 15 WHERE slug = 'net';

CREATE TABLE character_item_features_new (
    id INTEGER PRIMARY KEY,
    character_item_id INTEGER NOT NULL REFERENCES character_items(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    name_en TEXT NOT NULL DEFAULT '',
    name_ru TEXT NOT NULL DEFAULT '',
    stat TEXT NOT NULL DEFAULT 'note',
    value TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT 'common',
    source_url TEXT NOT NULL DEFAULT '',
    catalog_id INTEGER NOT NULL DEFAULT 0,
    catalog_kind TEXT NOT NULL DEFAULT '',
    catalog_slug TEXT NOT NULL DEFAULT ''
);

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 1, name_en, name_ru, 'atk_bonus', CAST(atk_bonus AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' WHEN catalog_slug != '' THEN 'magical' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE atk_bonus != 0 AND IFNULL(catalog_slug, '') != 'base-damage';

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 2, name_en, name_ru, 'dmg_bonus', CAST(dmg_bonus AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' WHEN catalog_slug != '' THEN 'magical' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE dmg_bonus != 0 AND IFNULL(catalog_slug, '') != 'base-damage';

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 3, name_en, name_ru, 'extra_dice', extra_damage,
       CASE WHEN source = 'catalog' THEN 'base' WHEN catalog_slug != '' THEN 'magical' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE extra_damage != '' AND IFNULL(catalog_slug, '') != 'base-damage';

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 4, name_en, name_ru, 'crit_range', CAST(crit_range AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' ELSE 'unique' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE crit_range != 0;

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 5, name_en, name_ru, 'ac_bonus', CAST(ac_bonus AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE ac_bonus != 0;

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 6, name_en, name_ru, 'ac_base', CAST(ac_base AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE ac_base != 0;

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 7, name_en, name_ru, 'ac_floor', CAST(ac_floor AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE ac_floor != 0;

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 8, name_en, name_ru, 'speed_bonus', CAST(speed_bonus AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE speed_bonus != 0;

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 9, name_en, name_ru, 'speed_mult', CAST(speed_mult AS TEXT),
       CASE WHEN source = 'catalog' THEN 'base' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE speed_mult != 0;

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 10, name_en, name_ru, 'personality',
       ('INT ' || int_score || ', WIS ' || wis_score || ', CHA ' || cha_score),
       'unique', source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features WHERE int_score != 0 OR wis_score != 0 OR cha_score != 0;

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 11, name_en, name_ru,
       CASE property_slug WHEN 'versatile' THEN 'versatile' WHEN 'thrown' THEN 'thrown' ELSE 'property' END,
       CASE
           WHEN property_slug = 'versatile' AND notes != '' THEN notes
           WHEN property_slug = 'thrown' AND notes != '' THEN notes
           WHEN property_slug != '' THEN property_slug
           ELSE notes
       END,
       CASE WHEN source = 'catalog' THEN 'base' ELSE 'common' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features
WHERE property_slug != '' AND IFNULL(catalog_slug, '') != 'base-damage';

INSERT INTO character_item_features_new (
    character_item_id, sort_order, name_en, name_ru, stat, value, origin, source_url, catalog_id, catalog_kind, catalog_slug
)
SELECT character_item_id, sort_order * 20 + 12, name_en, name_ru, 'note', notes,
       CASE WHEN catalog_slug != '' THEN 'magical' ELSE 'unique' END,
       source_url, catalog_id, catalog_kind, catalog_slug
FROM character_item_features
WHERE notes != '' AND property_slug = '' AND int_score = 0 AND wis_score = 0 AND cha_score = 0
  AND IFNULL(catalog_slug, '') != 'base-damage';

DROP TABLE character_item_features;
ALTER TABLE character_item_features_new RENAME TO character_item_features;
CREATE INDEX IF NOT EXISTS idx_item_features_item ON character_item_features(character_item_id);
