/**
 * Reward layer barrel (docs/05 §6). The Hero screen renders <RewardOverlay/>,
 * which drains the store's reward queue and plays each event.
 */

export { RewardOverlay, default } from './RewardOverlay';
export {
  RANK_META,
  CLASS_META,
  RARITY_META,
  SLOT_LABEL,
  rankFromLevel,
} from './labels';
export type { Meta } from './labels';
export { useReducedMotion } from './useReducedMotion';
