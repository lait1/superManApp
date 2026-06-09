-- 0003_appearance.sql — character appearance + onboarding flag.
-- Appearance ids reference the character asset manifest (docs/12,
-- cmd/genassets): bodyType a|b, skinTone s1..s4, hairstyle
-- bald|short|spiky|long|ponytail, hairColor dark|brown|blond|red.
-- Existing characters get the default look and onboarded=false, so they pass
-- through onboarding once and pick their own.

ALTER TABLE characters
  ADD COLUMN IF NOT EXISTS appearance JSONB NOT NULL
    DEFAULT '{"bodyType":"a","skinTone":"s2","hairstyle":"short","hairColor":"dark"}',
  ADD COLUMN IF NOT EXISTS onboarded BOOLEAN NOT NULL DEFAULT FALSE;
