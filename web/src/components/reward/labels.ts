/**
 * Shared display metadata for ranks / classes / rarity (docs/01 §4,§6, docs/04 §4,
 * docs/12 §7,§9). Single source for icon + Russian label + accent colour so the
 * Hero screen and the reward overlay render the same names everywhere.
 *
 * Lives in components/reward because the reward layer owns the rank-up / drop
 * presentation; the Hero screen imports from here too (same zone).
 */

import type { CharacterClass, Rank, Rarity, StatKey } from '../../types/api';

export interface Meta {
  icon: string;
  label: string;
  /** CSS colour token. */
  color: string;
}

/** Rank tiers — docs/01 §6 / docs/12 §7. */
export const RANK_META: Record<Rank, Meta> = {
  recruit: { icon: '🥉', label: 'Новобранец', color: 'var(--rarity-common)' },
  seeker: { icon: '🥈', label: 'Искатель', color: 'var(--tg-accent-text)' },
  veteran: { icon: '🥇', label: 'Ветеран', color: 'var(--rarity-legendary)' },
  master: { icon: '💎', label: 'Мастер', color: 'var(--rarity-rare)' },
  legend: { icon: '🔥', label: 'Легенда', color: 'var(--class-warrior)' },
};

/**
 * Stats — docs/01 §1 / docs/03. `short` is the 3-letter Russian abbreviation
 * for tight rows (StatBar, reward toasts); `label` is the full name.
 */
export const STAT_LABELS: Record<StatKey, { short: string; label: string }> = {
  STR: { short: 'СИЛ', label: 'Сила' },
  INT: { short: 'ИНТ', label: 'Интеллект' },
  DIS: { short: 'ДИС', label: 'Дисциплина' },
  VIT: { short: 'ЖИЗ', label: 'Жизненная сила' },
  CHA: { short: 'ХАР', label: 'Харизма' },
};

/** Classes / archetypes — docs/01 §4 / docs/12 §9. */
export const CLASS_META: Record<CharacterClass, Meta> = {
  warrior: { icon: '⚔️', label: 'Воин', color: 'var(--class-warrior)' },
  sage: { icon: '🧠', label: 'Мудрец', color: 'var(--class-sage)' },
  paladin: { icon: '🛡️', label: 'Паладин', color: 'var(--class-paladin)' },
  druid: { icon: '🌿', label: 'Друид', color: 'var(--class-druid)' },
  bard: { icon: '🎭', label: 'Бард', color: 'var(--class-bard)' },
  adventurer: { icon: '🧭', label: 'Авантюрист', color: 'var(--class-adventurer)' },
};

/**
 * Resolve a character level to its rank tier (docs/01 §6 / docs/12 §7).
 *   1–9 recruit · 10–24 seeker · 25–49 veteran · 50–99 master · 100+ legend
 * Used to translate the numeric from/to of RewardEvent.rankUp into rank names.
 */
export function rankFromLevel(level: number): Rank {
  if (level >= 100) return 'legend';
  if (level >= 50) return 'master';
  if (level >= 25) return 'veteran';
  if (level >= 10) return 'seeker';
  return 'recruit';
}

/** Item rarity — docs/04 §4. */
export const RARITY_META: Record<Rarity, Meta> = {
  common: { icon: '⚪', label: 'Обычный', color: 'var(--rarity-common)' },
  uncommon: { icon: '🟢', label: 'Необычный', color: 'var(--rarity-uncommon)' },
  rare: { icon: '🔵', label: 'Редкий', color: 'var(--rarity-rare)' },
  epic: { icon: '🟣', label: 'Эпический', color: 'var(--rarity-epic)' },
  legendary: { icon: '🟠', label: 'Легендарный', color: 'var(--rarity-legendary)' },
};

/** Equipment-slot Russian labels — docs/08 / docs/12 §5. */
export const SLOT_LABEL: Record<string, string> = {
  weapon: 'Оружие',
  armor: 'Броня',
  amulet: 'Амулет',
  head: 'Головной убор',
  boots: 'Обувь',
  back: 'Спина',
  aura: 'Аура',
  background: 'Фон',
  consumable: 'Расходник',
};
