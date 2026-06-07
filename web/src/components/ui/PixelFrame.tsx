/**
 * PixelFrame — the fixed-palette "scene panel" that frames pixel-art content
 * (docs/12 §2). Its colours are theme-independent so the sprite reads on both
 * light and dark Telegram themes. Used to wrap <CharacterCanvas/>.
 */

import type { CSSProperties, HTMLAttributes, ReactNode } from 'react';

export interface PixelFrameProps extends HTMLAttributes<HTMLDivElement> {
  /** Border thickness in px (chunky pixel border). Default 4. */
  borderWidth?: number;
  children?: ReactNode;
}

export function PixelFrame({
  borderWidth = 4,
  style,
  children,
  ...rest
}: PixelFrameProps) {
  const merged: CSSProperties = {
    position: 'relative',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background:
      'linear-gradient(180deg, var(--scene-bg) 0%, var(--scene-bg-2) 100%)',
    border: `${borderWidth}px solid var(--scene-border)`,
    boxShadow: 'inset 0 0 0 2px var(--scene-highlight)',
    borderRadius: 'var(--radius-md)',
    overflow: 'hidden',
    imageRendering: 'pixelated',
    ...style,
  };
  return (
    <div className="pixelated" style={merged} {...rest}>
      {children}
    </div>
  );
}
