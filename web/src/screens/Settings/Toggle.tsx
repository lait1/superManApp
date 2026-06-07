/**
 * Toggle — a themed on/off switch used by the notification settings block.
 * Animated with Framer Motion; no external dependency beyond the allowed set.
 */

import { motion } from 'framer-motion';
import type { CSSProperties } from 'react';

export interface ToggleProps {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  disabled?: boolean;
  id?: string;
}

const TRACK_W = 44;
const TRACK_H = 26;
const KNOB = 20;

export function Toggle({ checked, onChange, label, disabled = false, id }: ToggleProps) {
  const trackStyle: CSSProperties = {
    width: TRACK_W,
    height: TRACK_H,
    borderRadius: 'var(--radius-pill)',
    background: checked ? 'var(--tg-button)' : 'var(--tg-secondary-bg)',
    padding: 3,
    display: 'flex',
    alignItems: 'center',
    justifyContent: checked ? 'flex-end' : 'flex-start',
    transition: 'background var(--dur-base) var(--ease-standard)',
    flex: '0 0 auto',
  };

  return (
    <button
      type="button"
      id={id}
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      style={trackStyle}
    >
      <motion.span
        layout
        transition={{ type: 'spring', stiffness: 600, damping: 32 }}
        style={{
          width: KNOB,
          height: KNOB,
          borderRadius: '50%',
          background: 'var(--tg-button-text)',
          boxShadow: 'var(--shadow-1)',
          display: 'block',
        }}
      />
    </button>
  );
}
