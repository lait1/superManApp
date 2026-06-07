/**
 * Panel — a themed surface card (uses Telegram section background).
 * The general-purpose container for grouped content on a screen.
 */

import type { CSSProperties, HTMLAttributes, ReactNode } from 'react';

export interface PanelProps extends HTMLAttributes<HTMLDivElement> {
  /** Removes inner padding (for edge-to-edge content). */
  flush?: boolean;
  children?: ReactNode;
}

const baseStyle: CSSProperties = {
  background: 'var(--tg-section-bg)',
  borderRadius: 'var(--radius-lg)',
  boxShadow: 'var(--shadow-1)',
};

export function Panel({ flush = false, style, children, ...rest }: PanelProps) {
  const merged: CSSProperties = {
    ...baseStyle,
    padding: flush ? 0 : 'var(--space-4)',
    ...style,
  };
  return (
    <div style={merged} {...rest}>
      {children}
    </div>
  );
}
