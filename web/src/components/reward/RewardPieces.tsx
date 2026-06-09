/**
 * RewardPieces — the visual building blocks the RewardOverlay composes per
 * reward event (docs/05 §6, docs/12 §10). Each piece is a self-contained Framer
 * Motion sub-component; the overlay decides which ones to show for a given
 * RewardEvent and stacks them.
 *
 * Pieces:
 *   - XpGoldBadge   : the always-present "+N XP / +N 💰" badge (with crit styling)
 *   - CritFlash     : full-bleed gold flash + "CRIT ×N!" for isCrit
 *   - LevelUpBanner : LVL n→m flip with confetti + pulsing ring
 *   - RankUpOverlay : full-screen ceremonial rank transition
 *   - DropCard      : item card flying in, tinted by rarity
 *   - StreakChip    : 🔥 streak counter "click"
 *
 * All respect prefers-reduced-motion (via useReducedMotion + the global CSS
 * override). They never block input — the overlay layer handles tap-to-skip.
 */

import { motion } from 'framer-motion';
import type { CSSProperties } from 'react';
import type { Drop, LevelTransition, Reward } from '../../types/api';
import { Confetti } from './Confetti';
import { RANK_META, RARITY_META, SLOT_LABEL, STAT_LABELS } from './labels';
import { useReducedMotion } from './useReducedMotion';
import type { Rank } from '../../types/api';

const centerColumn: CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 'var(--space-3)',
  textAlign: 'center',
};

// ──────────────────────────────────────────────────────────────────────────
// +XP / +Gold badge (always present)
// ──────────────────────────────────────────────────────────────────────────

export interface XpGoldBadgeProps {
  reward: Reward;
}

export function XpGoldBadge({ reward }: XpGoldBadgeProps) {
  const reduced = useReducedMotion();
  const crit = reward.isCrit;

  return (
    <motion.div
      style={centerColumn}
      initial={reduced ? { opacity: 0 } : { opacity: 0, y: 24, scale: 0.85 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={reduced ? { opacity: 0 } : { opacity: 0, y: -28, scale: 0.9 }}
      transition={
        reduced
          ? { duration: 0.001 }
          : { type: 'spring', stiffness: 420, damping: 22 }
      }
    >
      <div
        className="text-pixel"
        style={{
          fontSize: crit ? 'var(--text-2xl)' : 'var(--text-xl)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
          color: crit ? 'var(--rarity-legendary)' : 'var(--tg-text)',
          textShadow: crit ? '0 0 14px var(--rarity-legendary)' : 'none',
        }}
      >
        +{reward.xp} XP
      </div>
      <div
        className="row"
        style={{
          gap: 'var(--space-3)',
          fontSize: 'var(--text-md)',
          color: 'var(--tg-subtitle-text)',
        }}
      >
        <span>💰 +{reward.gold}</span>
        <span style={{ color: 'var(--tg-hint)' }}>
          {STAT_LABELS[reward.statKey].short} +{reward.statPoints}
        </span>
      </div>
      {reward.capped && (
        <div style={{ fontSize: 'var(--text-xs)', color: 'var(--tg-hint)' }}>
          дневной лимит — награда снижена
        </div>
      )}
    </motion.div>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Crit flash
// ──────────────────────────────────────────────────────────────────────────

export interface CritFlashProps {
  multiplier: number;
}

export function CritFlash({ multiplier }: CritFlashProps) {
  const reduced = useReducedMotion();

  return (
    <>
      {!reduced && (
        <motion.div
          aria-hidden
          style={{
            position: 'absolute',
            inset: 0,
            background:
              'radial-gradient(circle at 50% 42%, var(--rarity-legendary), transparent 60%)',
            pointerEvents: 'none',
          }}
          initial={{ opacity: 0 }}
          animate={{ opacity: [0, 0.85, 0] }}
          transition={{ duration: 0.5, ease: 'easeOut' }}
        />
      )}
      <motion.div
        className="text-pixel"
        style={{
          fontSize: 'var(--text-xl)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
          color: 'var(--rarity-legendary)',
          textShadow: '0 0 18px var(--rarity-legendary)',
        }}
        initial={reduced ? { opacity: 0 } : { scale: 0.4, opacity: 0 }}
        animate={
          reduced
            ? { opacity: 1 }
            : { scale: [0.4, 1.25, 1], opacity: 1 }
        }
        transition={reduced ? { duration: 0.001 } : { duration: 0.45 }}
      >
        CRIT ×{multiplier}!
      </motion.div>
    </>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Level-up
// ──────────────────────────────────────────────────────────────────────────

export interface LevelUpBannerProps {
  transition: LevelTransition;
}

export function LevelUpBanner({ transition }: LevelUpBannerProps) {
  const reduced = useReducedMotion();

  return (
    <div style={{ ...centerColumn, position: 'relative', width: '100%' }}>
      {!reduced && <Confetti />}
      <motion.div
        style={{
          fontSize: 'var(--text-lg)',
          color: 'var(--tg-accent-text)',
          letterSpacing: '0.1em',
          textTransform: 'uppercase',
        }}
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: reduced ? 0 : 0.1 }}
      >
        Новый уровень!
      </motion.div>
      <motion.div
        className="text-pixel"
        style={{
          fontSize: 'var(--text-2xl)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
          color: 'var(--tg-text)',
        }}
        initial={reduced ? { opacity: 0 } : { scale: 0.5, opacity: 0 }}
        animate={
          reduced ? { opacity: 1 } : { scale: [0.5, 1.3, 1], opacity: 1 }
        }
        transition={
          reduced ? { duration: 0.001 } : { duration: 0.55, ease: 'backOut' }
        }
      >
        LVL {transition.from} → {transition.to}
      </motion.div>
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Rank-up (full-screen ceremonial)
// ──────────────────────────────────────────────────────────────────────────

export interface RankUpOverlayProps {
  /** from/to are level numbers in the API; we map to rank via the resolver. */
  fromRank: Rank;
  toRank: Rank;
}

export function RankUpOverlay({ fromRank, toRank }: RankUpOverlayProps) {
  const reduced = useReducedMotion();
  const to = RANK_META[toRank];
  const from = RANK_META[fromRank];

  return (
    <div style={{ ...centerColumn, position: 'relative', width: '100%' }}>
      {!reduced && (
        <motion.div
          aria-hidden
          style={{
            position: 'absolute',
            inset: '-40%',
            background: `radial-gradient(circle, ${to.color}, transparent 65%)`,
            pointerEvents: 'none',
          }}
          initial={{ opacity: 0, scale: 0.6 }}
          animate={{ opacity: [0, 0.6, 0.35], scale: 1 }}
          transition={{ duration: 0.9, ease: 'easeOut' }}
        />
      )}
      <div
        style={{
          fontSize: 'var(--text-md)',
          color: 'var(--tg-subtitle-text)',
          letterSpacing: '0.12em',
          textTransform: 'uppercase',
        }}
      >
        Повышение ранга
      </div>
      <motion.div
        style={{ fontSize: 'calc(var(--text-2xl) * 1.6)', lineHeight: 1 }}
        initial={reduced ? { opacity: 0 } : { scale: 0.3, rotate: -25, opacity: 0 }}
        animate={
          reduced ? { opacity: 1 } : { scale: [0.3, 1.25, 1], rotate: 0, opacity: 1 }
        }
        transition={reduced ? { duration: 0.001 } : { duration: 0.7, ease: 'backOut' }}
      >
        {to.icon}
      </motion.div>
      <div
        className="text-pixel"
        style={{
          fontSize: 'var(--text-xl)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
          color: to.color,
          textShadow: reduced ? 'none' : `0 0 16px ${to.color}`,
        }}
      >
        {to.label}
      </div>
      <div style={{ fontSize: 'var(--text-sm)', color: 'var(--tg-hint)' }}>
        {from.label} → {to.label}
      </div>
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Item drop card
// ──────────────────────────────────────────────────────────────────────────

export interface DropCardProps {
  drop: Drop;
}

export function DropCard({ drop }: DropCardProps) {
  const reduced = useReducedMotion();
  const meta = RARITY_META[drop.rarity];
  const slotLabel = SLOT_LABEL[drop.slot] ?? drop.slot;

  return (
    <motion.div
      style={{
        ...centerColumn,
        gap: 'var(--space-2)',
        padding: 'var(--space-5)',
        minWidth: 200,
        borderRadius: 'var(--radius-lg)',
        background: 'var(--tg-section-bg)',
        border: `2px solid ${meta.color}`,
        boxShadow: reduced ? 'var(--shadow-2)' : `0 0 24px ${meta.color}`,
      }}
      initial={
        reduced ? { opacity: 0 } : { opacity: 0, scale: 0.5, y: 40, rotate: -8 }
      }
      animate={{ opacity: 1, scale: 1, y: 0, rotate: 0 }}
      transition={
        reduced ? { duration: 0.001 } : { type: 'spring', stiffness: 260, damping: 20 }
      }
    >
      <div style={{ fontSize: 'var(--text-xs)', color: 'var(--tg-hint)' }}>
        🎁 Дроп!
      </div>
      <div style={{ fontSize: 'calc(var(--text-2xl) * 1.2)', lineHeight: 1 }}>
        {meta.icon}
      </div>
      <div
        style={{
          fontSize: 'var(--text-lg)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
          color: 'var(--tg-text)',
        }}
      >
        {drop.name}
      </div>
      <div style={{ fontSize: 'var(--text-sm)', color: meta.color }}>
        {meta.label} · {slotLabel}
      </div>
    </motion.div>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Streak chip
// ──────────────────────────────────────────────────────────────────────────

export interface StreakChipProps {
  days: number;
  mult: number;
}

export function StreakChip({ days, mult }: StreakChipProps) {
  const reduced = useReducedMotion();
  if (days <= 1) return null;

  return (
    <motion.div
      className="row"
      style={{
        gap: 'var(--space-2)',
        padding: '4px 12px',
        borderRadius: 'var(--radius-pill)',
        background: 'var(--tg-secondary-bg)',
        fontSize: 'var(--text-sm)',
      }}
      initial={reduced ? { opacity: 0 } : { opacity: 0, scale: 0.7 }}
      animate={reduced ? { opacity: 1 } : { opacity: 1, scale: [0.7, 1.15, 1] }}
      transition={reduced ? { duration: 0.001 } : { duration: 0.4 }}
    >
      <span aria-hidden>🔥</span>
      <span style={{ color: 'var(--class-warrior)', fontWeight: 700 }}>
        {days}
      </span>
      {mult > 1 && (
        <span style={{ color: 'var(--tg-hint)' }}>×{mult.toFixed(2)}</span>
      )}
    </motion.div>
  );
}
