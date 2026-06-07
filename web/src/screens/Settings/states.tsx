/**
 * Shared loading / error / empty state blocks for the Quests/Shop/Profile
 * screens. Themed via the UI-kit tokens; kept inside the screens zone.
 */

import type { ReactNode } from 'react';
import type { CSSProperties } from 'react';
import { Button, Panel } from '../../components/ui';
import type { ApiClientError } from '../../api/client';

const centerBox: CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  gap: 'var(--space-2)',
  padding: 'var(--space-6) var(--space-4)',
  textAlign: 'center',
};

/** Spinner-free, calm loading placeholder. */
export function LoadingState({ label = 'Загрузка…' }: { label?: string }) {
  return (
    <div className="muted" style={centerBox} role="status" aria-live="polite">
      {label}
    </div>
  );
}

/** Inline error with optional retry. */
export function ErrorState({
  error,
  onRetry,
}: {
  error: ApiClientError | Error | null;
  onRetry?: () => void;
}) {
  const message = error?.message || 'Что-то пошло не так';
  return (
    <Panel style={centerBox} role="alert">
      <div style={{ fontSize: 'var(--text-xl)' }} aria-hidden>
        ⚠️
      </div>
      <div style={{ fontWeight: 'var(--weight-bold)' as unknown as number }}>
        Ошибка загрузки
      </div>
      <div className="muted" style={{ fontSize: 'var(--text-sm)' }}>
        {message}
      </div>
      {onRetry && (
        <Button variant="secondary" size="sm" onClick={onRetry} style={{ marginTop: 'var(--space-2)' }}>
          Повторить
        </Button>
      )}
    </Panel>
  );
}

/** Friendly empty placeholder. */
export function EmptyState({ icon = '📭', title, hint }: { icon?: string; title: string; hint?: ReactNode }) {
  return (
    <div className="muted" style={centerBox}>
      <div style={{ fontSize: 'var(--text-2xl)' }} aria-hidden>
        {icon}
      </div>
      <div style={{ fontWeight: 'var(--weight-medium)' as unknown as number, color: 'var(--tg-text)' }}>
        {title}
      </div>
      {hint && <div style={{ fontSize: 'var(--text-sm)' }}>{hint}</div>}
    </div>
  );
}

/** Uppercase section header used across the screens. */
export function SectionHeader({ children }: { children: ReactNode }) {
  return (
    <div
      style={{
        fontSize: 'var(--text-xs)',
        fontWeight: 'var(--weight-bold)' as unknown as number,
        letterSpacing: '0.06em',
        textTransform: 'uppercase',
        color: 'var(--tg-section-header-text)',
        marginTop: 'var(--space-2)',
      }}
    >
      {children}
    </div>
  );
}
