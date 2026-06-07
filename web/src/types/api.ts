/**
 * API contract types — mirror the JSON described in docs/09-api.md.
 *
 * These are the single source of truth for request/response shapes shared
 * across the app. Do not redefine them in screens or components — import here.
 *
 * Base prefix: /api/v1. Auth via `Authorization: tma <initData>` or
 * `X-Device-Id: <uuid>` (dev fallback). Errors use the unified ErrorResponse.
 */

// ──────────────────────────────────────────────────────────────────────────
// Primitives / enums
// ──────────────────────────────────────────────────────────────────────────

/** Stat keys, see docs/03 & docs/08. */
export type StatKey = 'STR' | 'INT' | 'DIS' | 'VIT' | 'CHA';

/** Character class / archetype, see docs/12 §9. */
export type CharacterClass =
  | 'warrior'
  | 'sage'
  | 'paladin'
  | 'druid'
  | 'bard'
  | 'adventurer';

/** Rank tiers, see docs/12 §7. */
export type Rank = 'recruit' | 'seeker' | 'veteran' | 'master' | 'legend';

/** Equipment slots, see docs/08 shop_items.slot & docs/12 §5. */
export type EquipSlot =
  | 'weapon'
  | 'armor'
  | 'amulet'
  | 'head'
  | 'boots'
  | 'back'
  | 'aura'
  | 'background'
  | 'consumable';

/** Item rarity, see docs/04 / docs/08. */
export type Rarity = 'common' | 'uncommon' | 'rare' | 'epic' | 'legendary';

/** Quest lifecycle status, see docs/08 quest_progress.status. */
export type QuestStatus = 'active' | 'completed' | 'claimed' | 'expired';

/** Quest catalog type, see docs/08 quests.type. */
export type QuestType = 'daily' | 'weekly' | 'chain' | 'side' | 'class' | 'balance';

/** How an inventory item was acquired, see docs/08 inventory_items.acquired_via. */
export type AcquiredVia = 'purchase' | 'drop' | 'quest' | 'achievement';

/** Achievement category, see docs/08 achievements.category. */
export type AchievementCategory =
  | 'start'
  | 'streak'
  | 'volume'
  | 'level'
  | 'balance'
  | 'domain'
  | 'collection';

/** Notification kinds, see docs/06 / docs/09 settings. */
export type NotificationKind = 'daily' | 'streakReminder' | 'morning' | 'milestone';

// ──────────────────────────────────────────────────────────────────────────
// Errors (unified)
// ──────────────────────────────────────────────────────────────────────────

/** Unified error code set, see docs/09 §4 (400/401/404/409/429/500). */
export type ApiErrorCode =
  | 'bad_request'
  | 'unauthorized'
  | 'not_found'
  | 'insufficient_gold'
  | 'rate_limited'
  | 'internal'
  | (string & {});

export interface ApiError {
  code: ApiErrorCode;
  message: string;
}

/** `{ "error": { "code", "message" } }` */
export interface ErrorResponse {
  error: ApiError;
}

// ──────────────────────────────────────────────────────────────────────────
// Character & stats
// ──────────────────────────────────────────────────────────────────────────

/** Slot → equipped inventory item id (numeric). Partial: only equipped slots. */
export type EquippedMap = Partial<Record<EquipSlot, number>>;

/** Character block of GET /me. */
export interface Character {
  name: string;
  level: number;
  xpTotal: number;
  xpToNext: number;
  xpIntoLevel: number;
  gold: number;
  class: CharacterClass;
  rank: Rank;
  streakDays: number;
  streakMult: number;
  equipped: EquippedMap;
}

/** A single stat row in GET /me. */
export interface Stat {
  key: StatKey;
  value: number;
  level: number;
  intoLevel: number;
  toNext: number;
}

/** Compact character snapshot returned in a RewardEvent. */
export interface CharacterSnapshot {
  level: number;
  xpTotal: number;
  gold: number;
  streakDays: number;
}

/** GET /me */
export interface MeResponse {
  character: Character;
  stats: Stat[];
  /** activity keys checked in today, e.g. ["english", "gym"] */
  todayCheckins: string[];
}

// ──────────────────────────────────────────────────────────────────────────
// Check-in & reward events
// ──────────────────────────────────────────────────────────────────────────

/** POST /checkin request body. */
export interface CheckinRequest {
  activityKey: string;
  durationMin?: number;
  note?: string;
}

/** The core reward payload. */
export interface Reward {
  xp: number;
  gold: number;
  statKey: StatKey;
  statPoints: number;
  isCrit: boolean;
  streakDays: number;
  streakMult: number;
  /** present when the daily cap/cooldown reduced the reward */
  capped?: boolean;
}

/** Item drop description. */
export interface Drop {
  itemId: string;
  name: string;
  rarity: Rarity;
  slot: EquipSlot;
}

/** Generic from→to transition (level / rank). */
export interface LevelTransition {
  from: number;
  to: number;
}

/** Stat-level transition. */
export interface StatLevelTransition {
  key: StatKey;
  from: number;
  to: number;
}

/** Quest advancement entry inside a reward event. */
export interface QuestAdvanced {
  id: string;
  progress: number;
  target: number;
  status: QuestStatus;
}

/**
 * POST /checkin response — the reward event.
 * Fields `drop`, `levelUp`, `rankUp`, `statLevelUp` are null when no event.
 */
export interface RewardEvent {
  reward: Reward;
  drop: Drop | null;
  levelUp: LevelTransition | null;
  rankUp: LevelTransition | null;
  statLevelUp: StatLevelTransition | null;
  questsAdvanced: QuestAdvanced[];
  achievementsUnlocked: string[];
  character: CharacterSnapshot;
}

// ──────────────────────────────────────────────────────────────────────────
// Activities catalog
// ──────────────────────────────────────────────────────────────────────────

/** Item in GET /activities (catalog, see docs/08 activities). */
export interface Activity {
  key: string;
  title: string;
  statKey: StatKey;
  baseXp: number;
  baseGold: number;
  hasDuration: boolean;
  refMinutes?: number;
  rarity: Rarity;
  icon?: string;
  dailyCap: number;
}

export type ActivitiesResponse = Activity[];

// ──────────────────────────────────────────────────────────────────────────
// Quests
// ──────────────────────────────────────────────────────────────────────────

/** Reward shown on a quest card. */
export interface QuestReward {
  xp: number;
  gold: number;
  title?: string;
  item?: string;
}

/** A quest with progress (used in daily/weekly/chains). */
export interface Quest {
  id: string;
  title: string;
  progress: number;
  target: number;
  status: QuestStatus;
  reward: QuestReward;
}

/** GET /quests */
export interface QuestsResponse {
  daily: Quest[];
  weekly: Quest[];
  chains: Quest[];
}

/** POST /quests/{id}/claim */
export interface QuestClaimResponse {
  ok: boolean;
  gold: number;
  reward: QuestReward;
}

// ──────────────────────────────────────────────────────────────────────────
// Achievements
// ──────────────────────────────────────────────────────────────────────────

/** GET /achievements item. */
export interface Achievement {
  id: string;
  title: string;
  description?: string;
  category: AchievementCategory;
  icon?: string;
  unlocked: boolean;
  unlockedAt?: string;
}

export type AchievementsResponse = Achievement[];

// ──────────────────────────────────────────────────────────────────────────
// Shop & inventory
// ──────────────────────────────────────────────────────────────────────────

/** Item effect descriptor (e.g. { type, stat, value } or {} for cosmetics). */
export interface ItemEffect {
  type?: string;
  stat?: StatKey;
  value?: number;
}

/** GET /shop item (catalog, see docs/08 shop_items). */
export interface ShopItem {
  id: string;
  name: string;
  slot: EquipSlot;
  rarity: Rarity;
  /** null = not purchasable (drop/quest only) */
  price: number | null;
  effect: ItemEffect;
  purchasable: boolean;
  icon?: string;
}

export type ShopResponse = ShopItem[];

/** POST /shop/{itemId}/buy — success. */
export interface BuyResponse {
  ok: boolean;
  gold: number;
  inventoryItemId: number;
}

/** GET /inventory item (owned instance). */
export interface InventoryItem {
  id: number;
  shopItemId: string;
  name: string;
  slot: EquipSlot;
  rarity: Rarity;
  acquiredVia: AcquiredVia;
  quantity: number;
  icon?: string;
  equipped: boolean;
}

export type InventoryResponse = InventoryItem[];

/** Preview of stat deltas after equip, e.g. { INT: "+10% XP" }. */
export type StatsPreview = Partial<Record<StatKey, string>>;

/** POST /inventory/{id}/equip */
export interface EquipResponse {
  ok: boolean;
  equipped: EquippedMap;
  statsPreview: StatsPreview;
}

/** POST /inventory/{id}/unequip */
export interface UnequipResponse {
  ok: boolean;
  equipped: EquippedMap;
}

// ──────────────────────────────────────────────────────────────────────────
// Daily report
// ──────────────────────────────────────────────────────────────────────────

/** GET /report/today — in-app daily summary. */
export interface TodayReport {
  date: string;
  checkins: number;
  xpGained: number;
  goldGained: number;
  streakDays: number;
  statsGained: Partial<Record<StatKey, number>>;
}

// ──────────────────────────────────────────────────────────────────────────
// Notification settings
// ──────────────────────────────────────────────────────────────────────────

/** PUT /settings/notifications request & response body. */
export interface NotificationSettings {
  timezone: string;
  daily: boolean;
  streakReminder: boolean;
  morning: boolean;
  milestone: boolean;
  dailyHour: number;
}
