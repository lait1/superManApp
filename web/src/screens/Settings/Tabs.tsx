/**
 * Tabs — a small segmented control used by the Quests (Today/Week/Chains) and
 * Shop (Gear/Cosmetic/Consumable/Inventory) screens. Themed via UI-kit tokens.
 */

import type { CSSProperties } from 'react';
import { hapticFeedback } from '../../telegram/sdk';

export interface TabOption<T extends string> {
  key: T;
  label: string;
}

export interface TabsProps<T extends string> {
  options: ReadonlyArray<TabOption<T>>;
  value: T;
  onChange: (next: T) => void;
}

const wrapStyle: CSSProperties = {
  display: 'flex',
  gap: 'var(--space-1)',
  padding: 'var(--space-1)',
  background: 'var(--tg-secondary-bg)',
  borderRadius: 'var(--radius-pill)',
};

function tabStyle(active: boolean): CSSProperties {
  return {
    flex: 1,
    padding: '8px 6px',
    borderRadius: 'var(--radius-pill)',
    fontSize: 'var(--text-sm)',
    fontWeight: 'var(--weight-medium)' as unknown as number,
    textAlign: 'center',
    background: active ? 'var(--tg-button)' : 'transparent',
    color: active ? 'var(--tg-button-text)' : 'var(--tg-hint)',
    transition: 'background var(--dur-fast) var(--ease-standard)',
    whiteSpace: 'nowrap',
  };
}

export function Tabs<T extends string>({ options, value, onChange }: TabsProps<T>) {
  return (
    <div role="tablist" style={wrapStyle}>
      {options.map((opt) => {
        const active = opt.key === value;
        return (
          <button
            key={opt.key}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => {
              if (!active) {
                hapticFeedback('selection');
                onChange(opt.key);
              }
            }}
            style={tabStyle(active)}
          >
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
