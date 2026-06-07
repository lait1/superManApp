/**
 * CharacterCanvas — pixel-art paper-doll renderer (docs/12 §5, §12).
 *
 * Stacks PNG layers in the fixed z-order from manifest.layerOrder using absolute
 * positioning on the logical 128×192 canvas, then integer-scales the whole stack
 * with `image-rendering: pixelated` so pixels stay crisp (docs/12 §2, §12).
 *
 * Layer sourcing (docs/12 §5):
 *   scene      ← class
 *   auraBack   ← rank stage          (z1)
 *   back       ← equipped.back        (z2)
 *   body       ← class × rank stage   (z3, silhouette §8 + palette §9)
 *   boots      ← equipped.boots       (z4)
 *   armor      ← equipped.armor        (z5)
 *   arms       ← (folded into body placeholders) (z6)
 *   head/face  ← (folded into body)    (z7)
 *   headgear   ← equipped.head         (z8)
 *   weapon     ← equipped.weapon       (z9)
 *   amulet     ← equipped.amulet       (z10)
 *   auraFront  ← rank stage + particles for Master/Legend (z11)
 *   frame      ← rank stage            (z12)
 *
 * Idle "breathing" animation + a Master/Legend particle layer are driven by
 * Framer Motion and disabled under `prefers-reduced-motion` (docs/12 §10, §12).
 *
 * The `character` prop carries the live equip state (typically the cached
 * GET /me character); `overrides` lets the Character Lab force any combination
 * without a server round-trip.
 */

import { useMemo, type CSSProperties } from 'react';
import { motion, useReducedMotion } from 'framer-motion';
import type {
  Character,
  CharacterClass,
  EquipSlot,
  EquippedMap,
  Rank,
} from '../types/api';
import {
  assetUrl,
  findItem,
  rankToStage,
  resolveAuraFile,
  resolveBodyFile,
  resolveFrameFile,
  resolveSceneFile,
  useCharacterAssets,
  type CharacterManifest,
} from './useCharacterAssets';

/** Logical sprite canvas, docs/12 §2. */
export const SPRITE_WIDTH = 128;
export const SPRITE_HEIGHT = 192;

/**
 * Equip map keyed by string item ids — the renderer resolves files by item id.
 * (The API's EquippedMap keys slots to numeric inventory-item ids; for the
 * paper-doll we need the catalog item id, so the renderer accepts a string map
 * via `equippedItems` / `overrides`. Numeric `equipped` is accepted but ignored
 * unless a string map is supplied — see `resolveEquipped`.)
 */
export type EquippedItemMap = Partial<Record<EquipSlot, string>>;

/** Minimal character shape the renderer needs (class + rank + equip). */
export interface CharacterRenderInput {
  class: CharacterClass;
  rank: Rank;
  /** Slot → catalog item id (string). Drives which item PNGs are drawn. */
  equippedItems?: EquippedItemMap;
}

/** Lab/overrides — any field forces a value regardless of `character`. */
export interface CharacterOverrides {
  class?: CharacterClass;
  rank?: Rank;
  /** Explicit silhouette stage (1..5); wins over `rank` when set. */
  stage?: number;
  equippedItems?: EquippedItemMap;
}

export interface CharacterCanvasProps {
  /** Integer scale ×2…×4 (docs/12 §12). Defaults to the manifest scale. */
  scale?: number;
  /**
   * Live character (class/rank/equip). Usually the cached GET /me character.
   * Optional so the component can render a default recruit while loading.
   */
  character?: Character | CharacterRenderInput | null;
  /** Lab overrides — force any combination (docs/12 §12 quality tool). */
  overrides?: CharacterOverrides;

  // ── back-compat convenience props (Phase-1 callers) ──────────────────────
  /** Class shorthand; equivalent to `overrides.class`. */
  characterClass?: CharacterClass;
  /** Rank shorthand; equivalent to `overrides.rank`. */
  rank?: Rank;

  /** Disable the idle breathing/particle motion regardless of OS preference. */
  animate?: boolean;
  style?: CSSProperties;
  /** Accessible label for the rendered figure. */
  ariaLabel?: string;
}

/** Stages that get the live particle overlay (docs/12 §7: Master, Legend). */
const PARTICLE_STAGES = new Set<number>([4, 5]);

interface ResolvedLayer {
  key: string;
  src: string;
}

/**
 * A single absolutely-positioned PNG layer on the logical canvas. Sized to the
 * full canvas (every asset is drawn on 128×192 and pre-aligned to anchors), so
 * stacking them with `position:absolute; inset:0` reproduces the composition.
 */
function Layer({ src, z }: { src: string; z: number }) {
  return (
    <img
      src={src}
      alt=""
      aria-hidden
      draggable={false}
      className="pixelated"
      style={{
        position: 'absolute',
        inset: 0,
        width: '100%',
        height: '100%',
        zIndex: z,
        imageRendering: 'pixelated',
        pointerEvents: 'none',
      }}
    />
  );
}

/** Merge `character`, the shorthand props, and `overrides` into a concrete spec. */
function resolveSpec(
  manifest: CharacterManifest,
  props: CharacterCanvasProps,
): { cls: CharacterClass; stage: number; equipped: EquippedItemMap } {
  const { character, overrides, characterClass, rank } = props;

  const baseClass: CharacterClass =
    overrides?.class ?? characterClass ?? character?.class ?? 'adventurer';

  const baseRank: Rank = overrides?.rank ?? rank ?? character?.rank ?? 'recruit';

  const stage = overrides?.stage ?? rankToStage(baseRank, manifest);

  const equipped: EquippedItemMap = {
    ...resolveEquipped(character),
    ...(overrides?.equippedItems ?? {}),
  };

  return { cls: baseClass, stage, equipped };
}

/**
 * Pull a string-keyed equip map off the character input. The API `Character`
 * exposes `equipped` as slot→numeric id, which the paper-doll cannot resolve to
 * a file; callers that want items drawn pass `equippedItems` (slot→catalog id).
 */
function resolveEquipped(
  character: Character | CharacterRenderInput | null | undefined,
): EquippedItemMap {
  if (!character) return {};
  if ('equippedItems' in character && character.equippedItems) {
    return character.equippedItems;
  }
  return {};
}

/** Build the ordered list of resolved layers from the manifest + spec. */
function buildLayers(
  manifest: CharacterManifest,
  cls: CharacterClass,
  stage: number,
  equipped: EquippedItemMap,
): ResolvedLayer[] {
  const out: ResolvedLayer[] = [];

  const push = (key: string, file: string | undefined) => {
    if (file) out.push({ key, src: assetUrl(file) });
  };

  // Item slots that map onto manifest layer names.
  const slotForLayer: Partial<Record<string, EquipSlot>> = {
    back: 'back',
    boots: 'boots',
    armor: 'armor',
    headgear: 'head',
    weapon: 'weapon',
    amulet: 'amulet',
  };

  for (const layer of manifest.layerOrder) {
    switch (layer) {
      case 'scene':
        push(layer, resolveSceneFile(manifest, cls));
        break;
      case 'auraBack':
      case 'auraFront':
        push(layer, resolveAuraFile(manifest, stage));
        break;
      case 'body':
        push(layer, resolveBodyFile(manifest, cls, stage));
        break;
      case 'frame':
        push(layer, resolveFrameFile(manifest, stage));
        break;
      case 'arms':
      case 'head':
        // Folded into the body sprite in the placeholder pipeline (docs/12 §11):
        // no dedicated assets, nothing to draw.
        break;
      default: {
        const slot = slotForLayer[layer];
        if (slot) {
          const item = findItem(manifest, equipped[slot]);
          push(layer, item?.file);
        }
        break;
      }
    }
  }

  return out;
}

export function CharacterCanvas(props: CharacterCanvasProps) {
  const { scale, animate = true, style, ariaLabel } = props;
  const { data: manifest, isPending, isError } = useCharacterAssets();
  const reduceMotion = useReducedMotion();

  const intScale = Math.max(1, Math.round(scale ?? manifest?.scale ?? 3));

  const canvasW = manifest?.canvas.w ?? SPRITE_WIDTH;
  const canvasH = manifest?.canvas.h ?? SPRITE_HEIGHT;

  const spec = useMemo(
    () => (manifest ? resolveSpec(manifest, props) : null),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      manifest,
      props.character,
      props.overrides,
      props.characterClass,
      props.rank,
    ],
  );

  const layers = useMemo(
    () =>
      manifest && spec ? buildLayers(manifest, spec.cls, spec.stage, spec.equipped) : [],
    [manifest, spec],
  );

  const wrapperStyle: CSSProperties = {
    position: 'relative',
    width: canvasW * intScale,
    height: canvasH * intScale,
    imageRendering: 'pixelated',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    ...style,
  };

  // Motion: subtle idle "breathing" scale (docs/12 §10). Disabled when the user
  // prefers reduced motion or the caller opts out.
  const motionEnabled = animate && !reduceMotion;
  const breathe = motionEnabled
    ? { scale: [1, 1.012, 1], y: [0, -1, 0] }
    : undefined;

  const showParticles = motionEnabled && spec ? PARTICLE_STAGES.has(spec.stage) : false;

  if (isError || (!manifest && !isPending)) {
    // Manifest failed — render an accessible empty frame rather than crashing.
    return (
      <div
        className="pixelated"
        style={wrapperStyle}
        role="img"
        aria-label={ariaLabel ?? 'Персонаж недоступен'}
      />
    );
  }

  if (!manifest || !spec) {
    // Loading: reserve the exact box so layout does not jump.
    return (
      <div
        className="pixelated"
        style={wrapperStyle}
        role="img"
        aria-label={ariaLabel ?? 'Загрузка персонажа'}
        aria-busy="true"
      />
    );
  }

  return (
    <div
      className="pixelated"
      style={wrapperStyle}
      data-class={spec.cls}
      data-stage={spec.stage}
      role="img"
      aria-label={ariaLabel ?? `Персонаж: ${spec.cls}, стадия ${spec.stage}`}
    >
      <motion.div
        style={{
          position: 'relative',
          width: '100%',
          height: '100%',
          transformOrigin: '50% 90%',
        }}
        animate={breathe}
        transition={
          motionEnabled
            ? { duration: 3.2, repeat: Infinity, ease: 'easeInOut' }
            : undefined
        }
      >
        {layers.map((layer, i) => (
          <Layer key={layer.key} src={layer.src} z={i} />
        ))}

        {showParticles && (
          <ParticleField scale={intScale} count={spec.stage === 5 ? 10 : 6} />
        )}
      </motion.div>
    </div>
  );
}

/**
 * ParticleField — a lightweight front overlay of drifting pixel sparks for the
 * Master/Legend ranks (docs/12 §7, §11). Pure CSS-sized blocks animated by
 * Framer Motion; only mounted when motion is enabled.
 */
function ParticleField({ scale, count }: { scale: number; count: number }) {
  // Deterministic positions so the field is stable across renders.
  const sparks = useMemo(
    () =>
      Array.from({ length: count }, (_, i) => {
        const x = ((i * 53) % 100) / 100; // 0..1 across the canvas
        const delay = (i % count) * 0.45;
        const drift = 8 + ((i * 7) % 10);
        return { x, delay, drift };
      }),
    [count],
  );

  const dot = Math.max(2, Math.round(scale)); // particle size scales with sprite

  return (
    <div
      aria-hidden
      style={{
        position: 'absolute',
        inset: 0,
        zIndex: 999,
        pointerEvents: 'none',
        overflow: 'hidden',
      }}
    >
      {sparks.map((s, i) => (
        <motion.span
          key={i}
          style={{
            position: 'absolute',
            left: `${s.x * 100}%`,
            bottom: 0,
            width: dot,
            height: dot,
            background: 'var(--scene-highlight, #ffe9a8)',
            boxShadow: '0 0 4px rgba(255,233,168,0.9)',
            imageRendering: 'pixelated',
          }}
          initial={{ opacity: 0, y: 0 }}
          animate={{
            opacity: [0, 1, 0],
            y: [0, -s.drift * scale * 2],
          }}
          transition={{
            duration: 2.4,
            delay: s.delay,
            repeat: Infinity,
            ease: 'easeOut',
          }}
        />
      ))}
    </div>
  );
}

// Re-exported for callers that work with the API equip map directly.
export type { EquippedMap };
