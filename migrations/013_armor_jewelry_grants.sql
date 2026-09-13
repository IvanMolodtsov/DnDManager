-- Armor/jewelry builders plus parameterized grant pickers and DMG artifact templates.
-- PHB armor table and DMG artifact mechanics are encoded here (no 5e14 HTML scrape).

UPDATE catalog_items SET is_base = 1, source_url_ru = 'https://5e14.dnd.su/articles/inventory/95-armor-and-shields/'
WHERE slug IN (
    'padded-armor','leather-armor','studded-leather-armor','hide-armor','chain-shirt','scale-mail',
    'breastplate','half-plate-armor','ring-mail','chain-mail','splint-armor','plate-armor','shield'
);

UPDATE catalog_items SET name_ru = 'Стёганый доспех', armor_category = 'light', ac_base = 11, dex_max = -1, stealth_disadv = 1, str_min = 0, suggested_slot = 'armor' WHERE slug = 'padded-armor';
UPDATE catalog_items SET name_ru = 'Кожаный доспех', armor_category = 'light', ac_base = 11, dex_max = -1, stealth_disadv = 0, str_min = 0, suggested_slot = 'armor' WHERE slug = 'leather-armor';
UPDATE catalog_items SET name_ru = 'Проклёпанный кожаный доспех', armor_category = 'light', ac_base = 12, dex_max = -1, stealth_disadv = 0, str_min = 0, suggested_slot = 'armor' WHERE slug = 'studded-leather-armor';
UPDATE catalog_items SET name_ru = 'Шкурный доспех', armor_category = 'medium', ac_base = 12, dex_max = 2, stealth_disadv = 0, str_min = 0, suggested_slot = 'armor' WHERE slug = 'hide-armor';
UPDATE catalog_items SET name_ru = 'Кольчужная рубаха', armor_category = 'medium', ac_base = 13, dex_max = 2, stealth_disadv = 0, str_min = 0, suggested_slot = 'armor' WHERE slug = 'chain-shirt';
UPDATE catalog_items SET name_ru = 'Чешуйчатый доспех', armor_category = 'medium', ac_base = 14, dex_max = 2, stealth_disadv = 1, str_min = 0, suggested_slot = 'armor' WHERE slug = 'scale-mail';
UPDATE catalog_items SET name_ru = 'Кираса', armor_category = 'medium', ac_base = 14, dex_max = 2, stealth_disadv = 0, str_min = 0, suggested_slot = 'armor' WHERE slug = 'breastplate';
UPDATE catalog_items SET name_ru = 'Полулаты', armor_category = 'medium', ac_base = 15, dex_max = 2, stealth_disadv = 1, str_min = 0, suggested_slot = 'armor' WHERE slug = 'half-plate-armor';
UPDATE catalog_items SET name_ru = 'Колечный доспех', armor_category = 'heavy', ac_base = 14, dex_max = 0, stealth_disadv = 1, str_min = 0, suggested_slot = 'armor' WHERE slug = 'ring-mail';
UPDATE catalog_items SET name_ru = 'Кольчуга', armor_category = 'heavy', ac_base = 16, dex_max = 0, stealth_disadv = 1, str_min = 13, suggested_slot = 'armor' WHERE slug = 'chain-mail';
UPDATE catalog_items SET name_ru = 'Наборный доспех', armor_category = 'heavy', ac_base = 17, dex_max = 0, stealth_disadv = 1, str_min = 15, suggested_slot = 'armor' WHERE slug = 'splint-armor';
UPDATE catalog_items SET name_ru = 'Латы', armor_category = 'heavy', ac_base = 18, dex_max = 0, stealth_disadv = 1, str_min = 15, suggested_slot = 'armor' WHERE slug = 'plate-armor';
UPDATE catalog_items SET name_ru = 'Щит', armor_category = 'shield', ac_base = 2, dex_max = 0, stealth_disadv = 0, str_min = 0, suggested_slot = 'shield' WHERE slug = 'shield';

INSERT INTO catalog_items (
    id, slug, name_en, name_ru, kind, cost_gp, weight_lb, damage_dice, damage_type, armor_class,
    properties, rarity, source_url, source_url_ru, desc_en, armor_category, ac_base, dex_max,
    stealth_disadv, str_min, weapon_category, versatile_dice, range_normal, range_long,
    suggested_slot, requires_attunement, consumable, charges_max, is_stub, desc_ru, is_base
) VALUES
(2000001, 'ring', 'Ring', 'Кольцо', 'jewelry', 0, 0, '', '', '', '[]', '', '', '', '', '', 0, -1, 0, 0, '', '', 0, 0, 'ring', 0, 0, 0, 0, '', 1),
(2000002, 'neck', 'Amulet', 'Ожерелье', 'jewelry', 0, 0, '', '', '', '[]', '', '', '', '', '', 0, -1, 0, 0, '', '', 0, 0, 'neck', 0, 0, 0, 0, '', 1),
(2000003, 'cloak', 'Cloak', 'Плащ', 'jewelry', 0, 0, '', '', '', '[]', '', '', '', '', '', 0, -1, 0, 0, '', '', 0, 0, 'cloak', 0, 0, 0, 0, '', 1),
(2000004, 'head', 'Headwear', 'Головной убор', 'jewelry', 0, 0, '', '', '', '[]', '', '', '', '', '', 0, -1, 0, 0, '', '', 0, 0, 'head', 0, 0, 0, 0, '', 1),
(2000005, 'gloves', 'Gloves', 'Перчатки', 'jewelry', 0, 0, '', '', '', '[]', '', '', '', '', '', 0, -1, 0, 0, '', '', 0, 0, 'gloves', 0, 0, 0, 0, '', 1),
(2000006, 'boots', 'Boots', 'Сапоги', 'jewelry', 0, 0, '', '', '', '[]', '', '', '', '', '', 0, -1, 0, 0, '', '', 0, 0, 'boots', 0, 0, 0, 0, '', 1),
(2000007, 'belt', 'Belt', 'Пояс', 'jewelry', 0, 0, '', '', '', '[]', '', '', '', '', '', 0, -1, 0, 0, '', '', 0, 0, 'belt', 0, 0, 0, 0, '', 1);

INSERT INTO catalog_stat_features (id, slug, name_en, name_ru, stat, default_value, origin, source_url, sort_order) VALUES
(20, 'skill-proficiency', 'Skill proficiency', 'Владение навыком', 'skill_proficiency', 'stealth', 'magical', '', 300),
(21, 'resistance', 'Resistance', 'Сопротивление', 'resistance', 'fire', 'magical', '', 310),
(22, 'vulnerability', 'Vulnerability', 'Уязвимость', 'vulnerability', 'fire', 'unique', '', 320),
(23, 'immunity', 'Damage immunity', 'Иммунитет к урону', 'immunity', 'poison', 'magical', '', 330),
(24, 'condition-immunity', 'Condition immunity', 'Иммунитет к состоянию', 'condition_immunity', 'charmed', 'magical', '', 340),
(25, 'grant-cantrip', 'Cast a cantrip', 'Заговор', 'grant_cantrip', '', 'magical', '', 350),
(26, 'grant-spell-1', 'Cast a 1st-level spell', 'Заклинание 1 уровня', 'grant_spell', '', 'magical', '', 351),
(27, 'grant-spell-2', 'Cast a 2nd-level spell', 'Заклинание 2 уровня', 'grant_spell', '', 'magical', '', 352),
(28, 'grant-spell-3', 'Cast a 3rd-level spell', 'Заклинание 3 уровня', 'grant_spell', '', 'magical', '', 353),
(29, 'grant-spell-4', 'Cast a 4th-level spell', 'Заклинание 4 уровня', 'grant_spell', '', 'magical', '', 354),
(30, 'grant-spell-5', 'Cast a 5th-level spell', 'Заклинание 5 уровня', 'grant_spell', '', 'magical', '', 355),
(31, 'grant-spell-6', 'Cast a 6th-level spell', 'Заклинание 6 уровня', 'grant_spell', '', 'magical', '', 356),
(32, 'grant-spell-7', 'Cast a 7th-level spell', 'Заклинание 7 уровня', 'grant_spell', '', 'magical', '', 357),
(33, 'ability-bonus', 'Ability bonus', 'Бонус характеристики', 'ability_bonus', 'str:2', 'magical', '', 360),
(34, 'ability-penalty', 'Ability penalty', 'Штраф характеристики', 'ability_penalty', 'str:-1', 'unique', '', 370),
(35, 'speed-bonus', 'Speed bonus', 'Бонус скорости', 'speed_bonus', '10', 'magical', '', 380),
(40, 'artifact-ac-1', 'Artifact: +1 AC', 'Артефакт: +1 КД', 'ac_bonus', '1', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 400),
(41, 'artifact-skill', 'Artifact: skill proficiency', 'Артефакт: владение навыком', 'skill_proficiency', 'stealth', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 410),
(42, 'artifact-resistance', 'Artifact: resistance', 'Артефакт: сопротивление', 'resistance', 'fire', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 420),
(43, 'artifact-cantrip', 'Artifact: cantrip', 'Артефакт: заговор', 'grant_cantrip', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 430),
(44, 'artifact-spell-1', 'Artifact: 1st-level spell', 'Артефакт: заклинание 1 ур.', 'grant_spell', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 431),
(45, 'artifact-spell-2', 'Artifact: 2nd-level spell', 'Артефакт: заклинание 2 ур.', 'grant_spell', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 432),
(46, 'artifact-spell-3', 'Artifact: 3rd-level spell', 'Артефакт: заклинание 3 ур.', 'grant_spell', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 433),
(47, 'artifact-spell-4', 'Artifact: 4th-level spell', 'Артефакт: заклинание 4 ур.', 'grant_spell', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 434),
(48, 'artifact-spell-5', 'Artifact: 5th-level spell', 'Артефакт: заклинание 5 ур.', 'grant_spell', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 435),
(49, 'artifact-spell-6', 'Artifact: 6th-level spell', 'Артефакт: заклинание 6 ур.', 'grant_spell', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 436),
(50, 'artifact-spell-7', 'Artifact: 7th-level spell', 'Артефакт: заклинание 7 ур.', 'grant_spell', '', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 437),
(51, 'artifact-charm-immunity', 'Artifact: charm immunity', 'Артефакт: иммунитет к очарованию', 'condition_immunity', 'charmed', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 440),
(52, 'artifact-fear-immunity', 'Artifact: fear immunity', 'Артефакт: иммунитет к испугу', 'condition_immunity', 'frightened', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 441),
(53, 'artifact-disease-immunity', 'Artifact: disease immunity', 'Артефакт: иммунитет к болезням', 'immunity', 'disease', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 442),
(54, 'artifact-ability-plus-2', 'Artifact: +2 to an ability (max 24)', 'Артефакт: +2 к характеристике (макс. 24)', 'ability_bonus', 'str:2', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 450),
(55, 'artifact-extra-1d6', 'Artifact: extra 1d6 damage', 'Артефакт: доп. 1d6 урона', 'extra_dice', '1d6', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 460),
(56, 'artifact-speed-10', 'Artifact: +10 speed', 'Артефакт: +10 скорости', 'speed_bonus', '10', 'magical', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 470),
(57, 'artifact-regen', 'Artifact: regeneration', 'Артефакт: регенерация', 'note', 'While attuned, you regain 1d6 hit points at the start of your turn if you have at least 1 hit point.', 'unique', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 480),
(58, 'artifact-note-unattractive', 'Artifact: unattractive', 'Артефакт: непривлекательность', 'note', 'While attuned to this artifact, you have disadvantage on Charisma (Persuasion) checks made to socially impress others.', 'unique', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 500),
(59, 'artifact-note-glow', 'Artifact: glow', 'Артефакт: свечение', 'note', 'This artifact sheds dim light in a 5-foot radius while you are attuned to it. You cannot extinguish the light.', 'unique', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 510),
(60, 'artifact-note-whisper', 'Artifact: whispers', 'Артефакт: шёпот', 'note', 'While attuned, you hear occasional whispers in a language you do not know. The artifact has no mechanical effect beyond this flavor.', 'unique', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 520),
(61, 'artifact-note-hunger', 'Artifact: hunger', 'Артефакт: голод', 'note', 'While attuned, you must eat and drink twice as much as normal to avoid exhaustion from hunger or thirst.', 'unique', 'https://5e14.dnd.su/articles/inventory/139-artifacts/', 530);
