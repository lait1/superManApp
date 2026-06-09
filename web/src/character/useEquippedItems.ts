/**
 * useEquippedItems — bridges the API equip state to the paper-doll renderer.
 *
 * `character.equipped` maps slot → numeric inventory item id, but
 * <CharacterCanvas/> resolves sprites by CATALOG item id (string). This hook
 * joins the equipped map with the inventory (inventory id → shopItemId) so
 * screens can pass `overrides={{ equippedItems }}` and show worn gear.
 */

import { useMemo } from 'react';
import { useInventory } from '../lib/query';
import type { Character, EquipSlot } from '../types/api';
import type { EquippedItemMap } from './CharacterCanvas';

export function useEquippedItems(
  character: Pick<Character, 'equipped'> | null | undefined,
): EquippedItemMap {
  const { data: inventory } = useInventory();

  return useMemo(() => {
    const equipped = character?.equipped;
    if (!equipped || !inventory?.length) return {};
    const catalogIdByInvId = new Map(inventory.map((it) => [it.id, it.shopItemId]));
    const out: EquippedItemMap = {};
    for (const [slot, invId] of Object.entries(equipped)) {
      const catalogId = catalogIdByInvId.get(invId);
      if (catalogId) out[slot as EquipSlot] = catalogId;
    }
    return out;
  }, [character?.equipped, inventory]);
}
