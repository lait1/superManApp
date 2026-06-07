/**
 * useCharacterAssets — loads & caches the character asset manifest (docs/12 §12).
 *
 * The manifest is the contract written by `cmd/genassets` and consumed by
 * <CharacterCanvas/>: it lists the logical canvas size, integer scale, anchors,
 * the fixed z-order of layers, and the file names for every body (class×stage),
 * scene (class), aura/frame (stage) and equippable item (slot/rarity).
 *
 * Assets are static files under `web/public/assets/character/`, so we fetch the
 * manifest directly (NOT through the API client) and cache it with React Query.
 * It never changes during a session, so it is effectively immutable.
 */

import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { CharacterClass, EquipSlot, Rarity, Rank } from '../types/api';

/** Public base path for all generated character assets. */
export const CHARACTER_ASSET_BASE = '/assets/character';

/** `${CHARACTER_ASSET_BASE}/manifest.json` */
export const MANIFEST_URL = `${CHARACTER_ASSET_BASE}/manifest.json`;

/** Logical sprite canvas size, see docs/12 §2. */
export interface ManifestCanvas {
  w: number;
  h: number;
}

/**
 * Anchor points in canvas pixel coordinates (docs/12 §5). Point anchors are
 * `[x, y]` tuples; `feet_baseline` is a single `y`. Extra keys tolerated.
 */
export interface ManifestAnchors {
  head_center: [number, number];
  chest_center: [number, number];
  hand_right: [number, number];
  back_mount: [number, number];
  feet_baseline: number;
  [key: string]: [number, number] | number;
}

/** One equippable paper-doll layer (docs/12 §5). */
export interface ManifestItem {
  id: string;
  slot: EquipSlot;
  rarity: Rarity;
  file: string;
}

/** Stage index → file name (auras, frames). Keys are stage numbers as strings. */
export type StageFileMap = Record<string, string>;

/** Class → file name (scenes). */
export type ClassFileMap = Partial<Record<CharacterClass, string>>;

/** Class → stage → file name (bodies). */
export type BodyFileMap = Partial<Record<CharacterClass, StageFileMap>>;

/** The full manifest.json shape, matching the genassets contract exactly. */
export interface CharacterManifest {
  canvas: ManifestCanvas;
  scale: number;
  anchors: ManifestAnchors;
  /** Fixed bottom→top z-order of layers (docs/12 §5). */
  layerOrder: string[];
  classes: CharacterClass[];
  rankStages: number[];
  /** Stage number (as string) → rank name. */
  ranks: Record<string, Rank>;
  bodies: BodyFileMap;
  scenes: ClassFileMap;
  auras: StageFileMap;
  frames: StageFileMap;
  items: ManifestItem[];
}

/** React Query key for the manifest. */
export const CHARACTER_MANIFEST_QUERY_KEY = ['character', 'manifest'] as const;

async function fetchManifest(): Promise<CharacterManifest> {
  const res = await fetch(MANIFEST_URL, { headers: { Accept: 'application/json' } });
  if (!res.ok) {
    throw new Error(`Failed to load character manifest (${res.status})`);
  }
  return (await res.json()) as CharacterManifest;
}

/**
 * Loads & caches the character asset manifest. The manifest is treated as
 * immutable for the session (no refetch on focus/reconnect, infinite stale time).
 */
export function useCharacterAssets(): UseQueryResult<CharacterManifest, Error> {
  return useQuery<CharacterManifest, Error>({
    queryKey: CHARACTER_MANIFEST_QUERY_KEY,
    queryFn: fetchManifest,
    staleTime: Infinity,
    gcTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: 1,
  });
}

// ── helpers (pure, manifest-driven; reused by CharacterCanvas & Lab) ─────────

/** Map a Rank name to its 1-based silhouette stage (docs/12 §8). */
export function rankToStage(rank: Rank, manifest: CharacterManifest): number {
  for (const [stage, name] of Object.entries(manifest.ranks)) {
    if (name === rank) {
      const n = Number(stage);
      if (Number.isFinite(n)) return n;
    }
  }
  // Fallback: clamp to the first known stage.
  return manifest.rankStages[0] ?? 1;
}

/** Resolve the body sprite file for a class + stage, with sensible fallbacks. */
export function resolveBodyFile(
  manifest: CharacterManifest,
  cls: CharacterClass,
  stage: number,
): string | undefined {
  const byStage = manifest.bodies[cls];
  if (!byStage) return undefined;
  return byStage[String(stage)] ?? byStage[String(manifest.rankStages[0] ?? 1)];
}

/** Resolve the scene (background) file for a class. */
export function resolveSceneFile(
  manifest: CharacterManifest,
  cls: CharacterClass,
): string | undefined {
  return manifest.scenes[cls];
}

/** Resolve the aura file for a stage. */
export function resolveAuraFile(manifest: CharacterManifest, stage: number): string | undefined {
  return manifest.auras[String(stage)];
}

/** Resolve the frame file for a stage. */
export function resolveFrameFile(manifest: CharacterManifest, stage: number): string | undefined {
  return manifest.frames[String(stage)];
}

/** Find an item entry by its id. */
export function findItem(
  manifest: CharacterManifest,
  id: string | undefined,
): ManifestItem | undefined {
  if (!id) return undefined;
  return manifest.items.find((it) => it.id === id);
}

/** All items for a given slot (used by the Lab slot pickers). */
export function itemsForSlot(manifest: CharacterManifest, slot: EquipSlot): ManifestItem[] {
  return manifest.items.filter((it) => it.slot === slot);
}

/** Build an absolute public URL for an asset file name. */
export function assetUrl(file: string): string {
  return `${CHARACTER_ASSET_BASE}/${file}`;
}
