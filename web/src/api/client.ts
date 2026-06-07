/**
 * Typed fetch client for the superMen API (docs/09).
 *
 * - Base prefix: /api/v1
 * - Auth: `Authorization: tma <initData>` when inside Telegram, otherwise the
 *   dev fallback `X-Device-Id: <uuid>` (persisted in localStorage).
 * - Unified error parsing: non-2xx responses are parsed as ErrorResponse and
 *   thrown as ApiClientError.
 *
 * Import these functions from screens/hooks — do not call fetch directly.
 */

import { getInitData } from '../telegram/sdk';
import type {
  ActivitiesResponse,
  AchievementsResponse,
  ApiError,
  BuyResponse,
  CheckinRequest,
  EquipResponse,
  ErrorResponse,
  InventoryResponse,
  MeResponse,
  NotificationSettings,
  QuestClaimResponse,
  QuestsResponse,
  RewardEvent,
  ShopResponse,
  TodayReport,
  UnequipResponse,
} from '../types/api';

export const API_BASE = '/api/v1';

const DEVICE_ID_KEY = 'supermen.deviceId';

// ──────────────────────────────────────────────────────────────────────────
// Errors
// ──────────────────────────────────────────────────────────────────────────

/** Thrown for any non-2xx response (or transport failure). */
export class ApiClientError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, error: ApiError) {
    super(error.message);
    this.name = 'ApiClientError';
    this.status = status;
    this.code = error.code;
  }
}

function isErrorResponse(value: unknown): value is ErrorResponse {
  return (
    typeof value === 'object' &&
    value !== null &&
    'error' in value &&
    typeof (value as { error: unknown }).error === 'object' &&
    (value as { error: unknown }).error !== null
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Device id (dev fallback)
// ──────────────────────────────────────────────────────────────────────────

function generateUuid(): string {
  const cryptoObj = typeof crypto !== 'undefined' ? crypto : undefined;
  if (cryptoObj?.randomUUID) return cryptoObj.randomUUID();
  // RFC4122-ish fallback.
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/** Returns a stable device id, creating and persisting one if needed. */
export function getDeviceId(): string {
  if (typeof localStorage === 'undefined') return generateUuid();
  let id = localStorage.getItem(DEVICE_ID_KEY);
  if (!id) {
    id = generateUuid();
    localStorage.setItem(DEVICE_ID_KEY, id);
  }
  return id;
}

// ──────────────────────────────────────────────────────────────────────────
// Core request
// ──────────────────────────────────────────────────────────────────────────

function authHeaders(): Record<string, string> {
  const initData = getInitData();
  if (initData) {
    return { Authorization: `tma ${initData}` };
  }
  return { 'X-Device-Id': getDeviceId() };
}

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

interface RequestOptions {
  method?: HttpMethod;
  body?: unknown;
  signal?: AbortSignal;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal } = options;

  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...authHeaders(),
  };
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }

  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      signal,
    });
  } catch (cause) {
    throw new ApiClientError(0, {
      code: 'network_error',
      message: cause instanceof Error ? cause.message : 'network error',
    });
  }

  // 204 No Content.
  if (res.status === 204) {
    return undefined as T;
  }

  const raw: unknown = await res.json().catch(() => undefined);

  if (!res.ok) {
    if (isErrorResponse(raw)) {
      throw new ApiClientError(res.status, raw.error);
    }
    throw new ApiClientError(res.status, {
      code: 'http_error',
      message: `HTTP ${res.status}`,
    });
  }

  return raw as T;
}

// ──────────────────────────────────────────────────────────────────────────
// Endpoints (docs/09 §2)
// ──────────────────────────────────────────────────────────────────────────

export const api = {
  /** GET /me — character + stats + gold + streak. */
  getMe(signal?: AbortSignal): Promise<MeResponse> {
    return request<MeResponse>('/me', { signal });
  },

  /** POST /checkin — log an activity, returns the reward event. */
  checkin(body: CheckinRequest, signal?: AbortSignal): Promise<RewardEvent> {
    return request<RewardEvent>('/checkin', { method: 'POST', body, signal });
  },

  /** GET /activities — activity catalog. */
  getActivities(signal?: AbortSignal): Promise<ActivitiesResponse> {
    return request<ActivitiesResponse>('/activities', { signal });
  },

  /** GET /quests — quests with progress. */
  getQuests(signal?: AbortSignal): Promise<QuestsResponse> {
    return request<QuestsResponse>('/quests', { signal });
  },

  /** POST /quests/{id}/claim — claim a completed quest reward. */
  claimQuest(questId: string, signal?: AbortSignal): Promise<QuestClaimResponse> {
    return request<QuestClaimResponse>(`/quests/${encodeURIComponent(questId)}/claim`, {
      method: 'POST',
      signal,
    });
  },

  /** GET /achievements. */
  getAchievements(signal?: AbortSignal): Promise<AchievementsResponse> {
    return request<AchievementsResponse>('/achievements', { signal });
  },

  /** GET /shop — purchasable items. */
  getShop(signal?: AbortSignal): Promise<ShopResponse> {
    return request<ShopResponse>('/shop', { signal });
  },

  /** POST /shop/{itemId}/buy. */
  buyItem(itemId: string, signal?: AbortSignal): Promise<BuyResponse> {
    return request<BuyResponse>(`/shop/${encodeURIComponent(itemId)}/buy`, {
      method: 'POST',
      signal,
    });
  },

  /** GET /inventory. */
  getInventory(signal?: AbortSignal): Promise<InventoryResponse> {
    return request<InventoryResponse>('/inventory', { signal });
  },

  /** POST /inventory/{id}/equip. */
  equipItem(inventoryItemId: number, signal?: AbortSignal): Promise<EquipResponse> {
    return request<EquipResponse>(`/inventory/${inventoryItemId}/equip`, {
      method: 'POST',
      signal,
    });
  },

  /** POST /inventory/{id}/unequip. */
  unequipItem(inventoryItemId: number, signal?: AbortSignal): Promise<UnequipResponse> {
    return request<UnequipResponse>(`/inventory/${inventoryItemId}/unequip`, {
      method: 'POST',
      signal,
    });
  },

  /** GET /report/today — in-app daily summary. */
  getTodayReport(signal?: AbortSignal): Promise<TodayReport> {
    return request<TodayReport>('/report/today', { signal });
  },

  /** PUT /settings/notifications — toggles + timezone. */
  updateNotificationSettings(
    body: NotificationSettings,
    signal?: AbortSignal,
  ): Promise<NotificationSettings> {
    return request<NotificationSettings>('/settings/notifications', {
      method: 'PUT',
      body,
      signal,
    });
  },
};

export type ApiClient = typeof api;
