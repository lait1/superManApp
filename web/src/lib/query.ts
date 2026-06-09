/**
 * React Query setup + typed data hooks.
 *
 * `queryClient` is provided at the app root (src/main.tsx). The hooks below
 * wrap the typed API client (src/api/client.ts) — screens should use these
 * instead of calling the client directly, so caching/invalidation is shared.
 */

import {
  QueryClient,
  useMutation,
  useQuery,
  useQueryClient,
  type UseMutationResult,
  type UseQueryResult,
} from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import type {
  ActivitiesResponse,
  AchievementsResponse,
  BuyResponse,
  CharacterSetupRequest,
  CharacterSetupResponse,
  CheckinRequest,
  EquipResponse,
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

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      retry: (failureCount, error) => {
        // Don't retry auth / client errors.
        if (error instanceof ApiClientError && error.status >= 400 && error.status < 500) {
          return false;
        }
        return failureCount < 2;
      },
      refetchOnWindowFocus: false,
    },
  },
});

/** Stable query keys for cache reads/invalidations. */
export const queryKeys = {
  me: ['me'] as const,
  activities: ['activities'] as const,
  quests: ['quests'] as const,
  achievements: ['achievements'] as const,
  shop: ['shop'] as const,
  inventory: ['inventory'] as const,
  todayReport: ['report', 'today'] as const,
};

// ──────────────────────────────────────────────────────────────────────────
// Queries
// ──────────────────────────────────────────────────────────────────────────

export function useMe(): UseQueryResult<MeResponse, ApiClientError> {
  return useQuery({
    queryKey: queryKeys.me,
    queryFn: ({ signal }) => api.getMe(signal),
  });
}

export function useActivities(): UseQueryResult<ActivitiesResponse, ApiClientError> {
  return useQuery({
    queryKey: queryKeys.activities,
    queryFn: ({ signal }) => api.getActivities(signal),
    staleTime: 60 * 60_000, // catalog rarely changes
  });
}

export function useQuests(): UseQueryResult<QuestsResponse, ApiClientError> {
  return useQuery({
    queryKey: queryKeys.quests,
    queryFn: ({ signal }) => api.getQuests(signal),
  });
}

export function useAchievements(): UseQueryResult<AchievementsResponse, ApiClientError> {
  return useQuery({
    queryKey: queryKeys.achievements,
    queryFn: ({ signal }) => api.getAchievements(signal),
  });
}

export function useShop(): UseQueryResult<ShopResponse, ApiClientError> {
  return useQuery({
    queryKey: queryKeys.shop,
    queryFn: ({ signal }) => api.getShop(signal),
    staleTime: 10 * 60_000,
  });
}

export function useInventory(): UseQueryResult<InventoryResponse, ApiClientError> {
  return useQuery({
    queryKey: queryKeys.inventory,
    queryFn: ({ signal }) => api.getInventory(signal),
  });
}

export function useTodayReport(): UseQueryResult<TodayReport, ApiClientError> {
  return useQuery({
    queryKey: queryKeys.todayReport,
    queryFn: ({ signal }) => api.getTodayReport(signal),
  });
}

// ──────────────────────────────────────────────────────────────────────────
// Mutations
// ──────────────────────────────────────────────────────────────────────────

export function useSetupCharacter(): UseMutationResult<
  CharacterSetupResponse,
  ApiClientError,
  CharacterSetupRequest
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CharacterSetupRequest) => api.setupCharacter(body),
    onSuccess: (data) => {
      // Patch the cached /me so the onboarding gate opens without a refetch.
      qc.setQueryData<MeResponse>(queryKeys.me, (prev) =>
        prev ? { ...prev, character: data.character } : prev,
      );
      void qc.invalidateQueries({ queryKey: queryKeys.me });
    },
  });
}

export function useCheckin(): UseMutationResult<
  RewardEvent,
  ApiClientError,
  CheckinRequest
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CheckinRequest) => api.checkin(body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.me });
      void qc.invalidateQueries({ queryKey: queryKeys.quests });
      void qc.invalidateQueries({ queryKey: queryKeys.achievements });
    },
  });
}

export function useClaimQuest(): UseMutationResult<
  QuestClaimResponse,
  ApiClientError,
  string
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (questId: string) => api.claimQuest(questId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.quests });
      void qc.invalidateQueries({ queryKey: queryKeys.me });
    },
  });
}

export function useBuyItem(): UseMutationResult<BuyResponse, ApiClientError, string> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (itemId: string) => api.buyItem(itemId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.me });
      void qc.invalidateQueries({ queryKey: queryKeys.inventory });
      void qc.invalidateQueries({ queryKey: queryKeys.shop });
    },
  });
}

export function useEquipItem(): UseMutationResult<EquipResponse, ApiClientError, number> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (inventoryItemId: number) => api.equipItem(inventoryItemId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.me });
      void qc.invalidateQueries({ queryKey: queryKeys.inventory });
    },
  });
}

export function useUnequipItem(): UseMutationResult<
  UnequipResponse,
  ApiClientError,
  number
> {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (inventoryItemId: number) => api.unequipItem(inventoryItemId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.me });
      void qc.invalidateQueries({ queryKey: queryKeys.inventory });
    },
  });
}

export function useUpdateNotificationSettings(): UseMutationResult<
  NotificationSettings,
  ApiClientError,
  NotificationSettings
> {
  return useMutation({
    mutationFn: (body: NotificationSettings) => api.updateNotificationSettings(body),
  });
}
