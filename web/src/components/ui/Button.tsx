/**
 * Button — primary action button styled with the Telegram theme button colour.
 * Variants: primary (filled), secondary (subtle), ghost (text-only).
 */

import type { ButtonHTMLAttributes, CSSProperties, ReactNode } from 'react';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost';
export type ButtonSize = 'sm' | 'md' | 'lg';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  children?: ReactNode;
}

const sizePadding: Record<ButtonSize, string> = {
  sm: '6px 12px',
  md: '10px 16px',
  lg: '14px 20px',
};

const sizeFont: Record<ButtonSize, string> = {
  sm: 'var(--text-sm)',
  md: 'var(--text-md)',
  lg: 'var(--text-lg)',
};

function variantStyle(variant: ButtonVariant): CSSProperties {
  switch (variant) {
    case 'secondary':
      return {
        background: 'var(--tg-secondary-bg)',
        color: 'var(--tg-text)',
      };
    case 'ghost':
      return {
        background: 'transparent',
        color: 'var(--tg-link)',
      };
    case 'primary':
    default:
      return {
        background: 'var(--tg-button)',
        color: 'var(--tg-button-text)',
      };
  }
}

export function Button({
  variant = 'primary',
  size = 'md',
  fullWidth = false,
  style,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  const merged: CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 'var(--space-2)',
    width: fullWidth ? '100%' : undefined,
    padding: sizePadding[size],
    fontSize: sizeFont[size],
    fontWeight: 'var(--weight-medium)' as unknown as number,
    borderRadius: 'var(--radius-md)',
    transition: `transform var(--dur-fast) var(--ease-standard), opacity var(--dur-fast) var(--ease-standard)`,
    ...variantStyle(variant),
    ...style,
  };
  return (
    <button type={type} style={merged} {...rest}>
      {children}
    </button>
  );
}
