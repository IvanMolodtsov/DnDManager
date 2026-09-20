-- Fill PHB dice for spells 5eapi leaves unstructured (or omits entirely).
-- Does not scrape dnd.su article text.

UPDATE catalog_spells SET damage_formula = '1d6', damage_type = 'necrotic' WHERE slug = 'hex' AND damage_formula = '';
UPDATE catalog_spells SET damage_formula = '1d6' WHERE slug = 'hunters-mark' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '1d12', damage_type = 'lightning', scale_kind = 'slot',
    damage_at_slot = '{"1":"1d12","2":"2d12","3":"3d12","4":"4d12","5":"5d12","6":"6d12","7":"7d12","8":"8d12","9":"9d12"}'
WHERE slug = 'witch-bolt' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '3d8', scale_kind = 'slot',
    damage_at_slot = '{"1":"3d8","2":"4d8","3":"5d8","4":"6d8","5":"7d8","6":"8d8","7":"9d8","8":"10d8","9":"11d8"}'
WHERE slug = 'chromatic-orb' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '5', damage_type = 'cold', scale_kind = 'slot',
    damage_at_slot = '{"1":"5","2":"10","3":"15","4":"20","5":"25","6":"30","7":"35","8":"40","9":"45"}'
WHERE slug = 'armor-of-agathys' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '2d6', damage_type = 'necrotic', scale_kind = 'slot',
    damage_at_slot = '{"1":"2d6","2":"3d6","3":"4d6","4":"5d6","5":"6d6","6":"7d6","7":"8d6","8":"9d6","9":"10d6"}'
WHERE slug = 'arms-of-hadar' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '3d6', damage_type = 'psychic', scale_kind = 'slot',
    damage_at_slot = '{"1":"3d6","2":"4d6","3":"5d6","4":"6d6","5":"7d6","6":"8d6","7":"9d6","8":"10d6","9":"11d6"}'
WHERE slug = 'dissonant-whispers' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '1d10', damage_type = 'piercing', scale_kind = 'slot',
    damage_at_slot = '{"1":"1d10","2":"2d10","3":"3d10","4":"4d10","5":"5d10","6":"6d10","7":"7d10","8":"8d10","9":"9d10"}'
WHERE slug = 'hail-of-thorns' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '1d6', damage_type = 'piercing', scale_kind = 'slot',
    damage_at_slot = '{"1":"1d6","2":"2d6","3":"3d6","4":"4d6","5":"5d6","6":"6d6","7":"7d6","8":"8d6","9":"9d6"}'
WHERE slug = 'ensnaring-strike' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '1d6', damage_type = 'piercing', scale_kind = 'character',
    damage_at_character = '{"1":"1d6","5":"2d6","11":"3d6","17":"4d6"}'
WHERE slug = 'thorn-whip' AND damage_formula = '';
UPDATE catalog_spells SET damage_formula = '2d6', damage_type = 'thunder' WHERE slug = 'thunderous-smite' AND damage_formula = '';
UPDATE catalog_spells SET damage_formula = '1d6', damage_type = 'psychic' WHERE slug = 'wrathful-smite' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '1d6', damage_type = 'fire', scale_kind = 'slot',
    damage_at_slot = '{"1":"1d6","2":"2d6","3":"3d6","4":"4d6","5":"5d6","6":"6d6","7":"7d6","8":"8d6","9":"9d6"}'
WHERE slug = 'searing-smite' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '2d6', damage_type = 'radiant', scale_kind = 'slot',
    damage_at_slot = '{"2":"2d6","3":"3d6","4":"4d6","5":"5d6","6":"6d6","7":"7d6","8":"8d6","9":"9d6"}'
WHERE slug = 'branding-smite' AND damage_formula = '';
UPDATE catalog_spells SET damage_formula = '5d10', damage_type = 'force' WHERE slug = 'banishing-smite' AND damage_formula = '';
UPDATE catalog_spells SET damage_formula = '4d6', damage_type = 'psychic' WHERE slug = 'staggering-smite' AND damage_formula = '';
UPDATE catalog_spells SET damage_formula = '3d8', damage_type = 'radiant' WHERE slug = 'blinding-smite' AND damage_formula = '';
UPDATE catalog_spells SET
    damage_formula = '1d4', scale_kind = 'slot',
    damage_at_slot = '{"3":"1d4","5":"2d4","7":"3d4"}'
WHERE slug = 'elemental-weapon' AND damage_formula = '';
UPDATE catalog_spells SET damage_formula = '2d6', damage_type = 'cold' WHERE slug = 'hunger-of-hadar' AND damage_formula = '';
