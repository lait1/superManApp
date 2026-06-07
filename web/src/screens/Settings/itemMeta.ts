/**
 * Shared item/quest presentation helpers for the Quests/Shop/Profile screens.
 *
 * Kept inside the screens zone (Settings dir) and imported by sibling screens so
 * rarity colours, slot labels and section grouping stay consistent. Maps mirror
 * docs/04 §4 (rarity) and docs/04 §3 / docs/08 (equip slots).
 */

import type { EquipSlot, Rarity } from '../../types/api';

/** Rarity → CSS colour token (docs/04 §4). */
export const RARITY_COLOR: Record<Rarity, string> = {
  common: 'var(--rarity-common)',
  uncommon: 'var(--rarity-uncommon)',
  rare: 'var(--rarity-rare)',
  epic: 'var(--rarity-epic)',
  legendary: 'var(--rarity-legendary)',
};

/** Rarity → Russian label. */
export const RARITY_LABEL: Record<Rarity, string> = {
  common: 'Обычный',
  uncommon: 'Необычный',
  rare: 'Редкий',
  epic: 'Эпический',
  legendary: 'Легендарный',
};

/** Equip slot → emoji icon (docs/04 §3). */
export const SLOT_ICON: Record<EquipSlot, string> = {
  weapon: '⚔️',
  armor: '🛡️',
  amulet: '📿',
  head: '🪖',
  boots: '🥾',
  back: '🧥',
  aura: '🎯',
  background: '🖼️',
  consumable: '🧪',
};

/** Equip slot → Russian label. */
export const SLOT_LABEL: Record<EquipSlot, string> = {
  weapon: 'Оружие',
  armor: 'Броня',
  amulet: 'Амулет',
  head: 'Голова',
  boots: 'Обувь',
  back: 'Накидка',
  aura: 'Аура',
  background: 'Фон',
  consumable: 'Расходник',
};

/** Shop sections (docs/04 §7). */
export type ShopSection = 'gear' | 'cosmetic' | 'consumable';

/** Map a slot to its shop section. */
export function slotSection(slot: EquipSlot): ShopSection {
  switch (slot) {
    case 'consumable':
      return 'consumable';
    case 'background':
    case 'aura':
    case 'back':
      return 'cosmetic';
    case 'weapon':
    case 'armor':
    case 'amulet':
    case 'head':
    case 'boots':
    default:
      return 'gear';
  }
}

/** Resolve an icon for an item: explicit icon wins, else the slot default. */
export function itemIcon(slot: EquipSlot, icon?: string): string {
  return icon && icon.length > 0 ? icon : SLOT_ICON[slot];
}

/** Format a gold amount with thin-space grouping (e.g. 6 420). */
export function formatGold(value: number): string {
  return value.toLocaleString('ru-RU');
}
