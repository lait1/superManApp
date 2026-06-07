/**
 * Confetti — lightweight Framer-Motion particle burst for level-up / rank-up
 * (docs/05 §6: "level-up → конфетти 🎉"). No extra deps: a fixed set of small
 * coloured squares animate outward + fall, then fade. Pixel-art flavour via
 * square chips and the fixed rarity/class palette.
 *
 * Suppressed entirely under prefers-reduced-motion (caller checks first, but we
 * also guard here defensively).
 */

import { motion } from 'framer-motion';
import { useMemo } from 'react';
import { useReducedMotion } from './useReducedMotion';

export interface ConfettiProps {
  /** Number of particles. Default 28. */
  count?: number;
  /** Colours to sample from. */
  colors?: string[];
}

interface Piece {
  id: number;
  x: number; // horizontal drift target (px)
  y: number; // vertical fall target (px)
  rotate: number;
  delay: number;
  size: number;
  color: string;
}

const FALLBACK_COLOR = 'var(--tg-accent-text)';

const DEFAULT_COLORS: string[] = [
  'var(--rarity-legendary)',
  'var(--rarity-epic)',
  'var(--rarity-rare)',
  'var(--rarity-uncommon)',
  'var(--class-bard)',
  FALLBACK_COLOR,
];

function rand(min: number, max: number): number {
  return min + Math.random() * (max - min);
}

export function Confetti({ count = 28, colors = DEFAULT_COLORS }: ConfettiProps) {
  const reduced = useReducedMotion();

  const pieces = useMemo<Piece[]>(() => {
    const palette = colors.length > 0 ? colors : DEFAULT_COLORS;
    return Array.from({ length: count }, (_, id) => ({
      id,
      x: rand(-160, 160),
      y: rand(120, 320),
      rotate: rand(-220, 220),
      delay: rand(0, 0.18),
      size: rand(6, 12),
      color: palette[id % palette.length] ?? FALLBACK_COLOR,
    }));
  }, [count, colors]);

  if (reduced) return null;

  return (
    <div
      aria-hidden
      style={{
        position: 'absolute',
        inset: 0,
        overflow: 'hidden',
        pointerEvents: 'none',
      }}
    >
      {pieces.map((p) => (
        <motion.span
          key={p.id}
          style={{
            position: 'absolute',
            top: '38%',
            left: '50%',
            width: p.size,
            height: p.size,
            background: p.color,
            borderRadius: 1,
          }}
          initial={{ x: 0, y: 0, opacity: 1, rotate: 0, scale: 1 }}
          animate={{
            x: p.x,
            y: p.y,
            rotate: p.rotate,
            opacity: [1, 1, 0],
            scale: [1, 1, 0.6],
          }}
          transition={{
            duration: 1.1,
            delay: p.delay,
            ease: [0.2, 0.6, 0.3, 1],
          }}
        />
      ))}
    </div>
  );
}
