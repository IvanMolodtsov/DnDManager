-- Status rows: source, duration, and display formulas. Temp HP stacks across sources (house rule).

ALTER TABLE character_effects ADD COLUMN source TEXT NOT NULL DEFAULT 'spell';
ALTER TABLE character_effects ADD COLUMN duration_key TEXT NOT NULL DEFAULT '';
ALTER TABLE character_effects ADD COLUMN formula_en TEXT NOT NULL DEFAULT '';
ALTER TABLE character_effects ADD COLUMN formula_ru TEXT NOT NULL DEFAULT '';
