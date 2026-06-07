/**
 * ProgressBar — animated fill bar (docs/05 §6: always animate fill, never jump).
 * Generic; StatBar and the XP bar build on top of it.
 */

import { motion } from 'framer-motion';
import type { CSSProperties } from 'react';

export interface ProgressBarProps {
  /** Current value. */
  value: number;
  /** Max value (target). */
  max: number;
  /** Bar fill colour (CSS var or literal). Default: accent. */
  color?: string;
  /** Track height in px. Default 10. */
  height?: number;
  /** Optional label rendered above/inside the bar by the caller instead. */
  className?: string;
  style?: CSSProperties;
}

export function ProgressBar({
  value,
  max,
  color = 'var(--tg-button)',
  height = 10,
  className,
  style,
}: ProgressBarProps) {
  const safeMax = max > 0 ? max : 1;
  const pct = Math.max(0, Math.min(100, (value / safeMax) * 100));

  const trackStyle: CSSProperties = {
    width: '100%',
    height,
    borderRadius: 'var(--radius-pill)',
    background: 'var(--tg-secondary-bg)',
    overflow: 'hidden',
    ...style,
  };

  return (
    <div
      className={className}
      style={trackStyle}
      role="progressbar"
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={max}
    >
      <motion.div
        initial={false}
        animate={{ width: `${pct}%` }}
        transition={{
          duration: 0.5,
          ease: [0.2, 0, 0, 1],
        }}
        style={{
          height: '100%',
          background: color,
          borderRadius: 'var(--radius-pill)',
        }}
      />
    </div>
  );
}
