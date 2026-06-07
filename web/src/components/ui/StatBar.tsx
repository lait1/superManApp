/**
 * StatBar — one row of the Hero screen stat list (docs/05 §2):
 *   icon · key · animated progress to next stat level · LVL n
 * Built on ProgressBar; colour comes from the per-stat accent token.
 */

import type { CSSProperties } from 'react';
import type { StatKey } from '../../types/api';
import { ProgressBar } from './ProgressBar';

export interface StatBarProps {
  statKey: StatKey;
  /** Progress into the current stat level. */
  intoLevel: number;
  /** Points needed to reach the next level. */
  toNext: number;
  level: number;
  style?: CSSProperties;
}

const STAT_META: Record<StatKey, { icon: string; color: string }> = {
  STR: { icon: '💪', color: 'var(--stat-str)' },
  INT: { icon: '🧠', color: 'var(--stat-int)' },
  DIS: { icon: '🎯', color: 'var(--stat-dis)' },
  VIT: { icon: '❤️', color: 'var(--stat-vit)' },
  CHA: { icon: '✨', color: 'var(--stat-cha)' },
};

export function StatBar({ statKey, intoLevel, toNext, level, style }: StatBarProps) {
  const meta = STAT_META[statKey];
  const max = intoLevel + toNext;

  const rowStyle: CSSProperties = {
    display: 'grid',
    gridTemplateColumns: 'auto 2.5em 1fr auto',
    alignItems: 'center',
    gap: 'var(--space-2)',
    ...style,
  };

  return (
    <div style={rowStyle}>
      <span aria-hidden style={{ fontSize: 'var(--text-md)' }}>
        {meta.icon}
      </span>
      <span
        style={{
          fontSize: 'var(--text-sm)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
          color: 'var(--tg-hint)',
        }}
      >
        {statKey}
      </span>
      <ProgressBar value={intoLevel} max={max} color={meta.color} height={8} />
      <span
        style={{
          fontSize: 'var(--text-xs)',
          color: 'var(--tg-hint)',
          whiteSpace: 'nowrap',
        }}
      >
        LVL {level}
      </span>
    </div>
  );
}
