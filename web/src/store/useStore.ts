/**
 * Global UI/state store (zustand).
 *
 * Holds:
 *  - `character`: latest known character snapshot (cached from GET /me) so the
 *    Hero screen and tab bar can read it synchronously.
 *  - check-in modal open/close state (the modal is rendered above routes).
 *  - the reward-event queue: each successful check-in pushes a RewardEvent that
 *    the reward animation layer drains one-by-one (docs/05 §6).
 *
 * Server data fetching/caching lives in React Query (src/lib/query.ts). This
 * store is for cross-cutting UI state and the optimistic reward queue.
 */

import { create } from 'zustand';
import type { Character, RewardEvent } from '../types/api';

export interface StoreState {
  // ── character snapshot ───────────────────────────────────────────────────
  character: Character | null;
  setCharacter: (character: Character | null) => void;

  // ── check-in modal ─────────────────────────────────────────────────────────
  checkinOpen: boolean;
  openCheckin: () => void;
  closeCheckin: () => void;
  toggleCheckin: () => void;

  // ── reward-event queue ─────────────────────────────────────────────────────
  rewardQueue: RewardEvent[];
  enqueueReward: (event: RewardEvent) => void;
  /** Removes and returns the head of the queue (or null if empty). */
  dequeueReward: () => RewardEvent | null;
  clearRewards: () => void;
}

export const useStore = create<StoreState>((set, get) => ({
  character: null,
  setCharacter: (character) => set({ character }),

  checkinOpen: false,
  openCheckin: () => set({ checkinOpen: true }),
  closeCheckin: () => set({ checkinOpen: false }),
  toggleCheckin: () => set((state) => ({ checkinOpen: !state.checkinOpen })),

  rewardQueue: [],
  enqueueReward: (event) =>
    set((state) => ({ rewardQueue: [...state.rewardQueue, event] })),
  dequeueReward: () => {
    const [head, ...rest] = get().rewardQueue;
    if (head === undefined) return null;
    set({ rewardQueue: rest });
    return head;
  },
  clearRewards: () => set({ rewardQueue: [] }),
}));

// Convenience selectors (stable references, avoid re-renders).
export const selectCharacter = (s: StoreState): Character | null => s.character;
export const selectCheckinOpen = (s: StoreState): boolean => s.checkinOpen;
export const selectRewardQueue = (s: StoreState): RewardEvent[] => s.rewardQueue;
