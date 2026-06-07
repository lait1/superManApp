/**
 * NotificationSettings — the in-profile notification controls (docs/06 §6).
 *
 * Toggles for daily report / streak reminder / morning quests / milestones,
 * plus the daily-report hour and the detected timezone. Persists via
 * PUT /settings/notifications (useUpdateNotificationSettings).
 *
 * The API has no GET for current settings, so the form is seeded with sensible
 * defaults (docs/06 §2 slots) and the browser timezone, then driven locally.
 */

import { useState } from 'react';
import type { CSSProperties } from 'react';
import { Button, Panel } from '../../components/ui';
import { useUpdateNotificationSettings } from '../../lib/query';
import { getTimezone, hapticFeedback } from '../../telegram/sdk';
import type { NotificationSettings as Settings } from '../../types/api';
import { Toggle } from './Toggle';

function defaultSettings(): Settings {
  return {
    timezone: getTimezone(),
    daily: true,
    streakReminder: true,
    morning: false,
    milestone: true,
    dailyHour: 21,
  };
}

const rowStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 'var(--space-3)',
  minHeight: 36,
};

const HOURS: readonly number[] = Array.from({ length: 24 }, (_, h) => h);

function formatHour(h: number): string {
  return `${String(h).padStart(2, '0')}:00`;
}

export default function NotificationSettings() {
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const mutation = useUpdateNotificationSettings();

  function patch<K extends keyof Settings>(key: K, value: Settings[K]): void {
    setSettings((prev) => ({ ...prev, [key]: value }));
  }

  function onToggle<K extends keyof Settings>(key: K, value: boolean): void {
    hapticFeedback('selection');
    patch(key, value as Settings[K]);
  }

  function onSave(): void {
    mutation.mutate(settings, {
      onSuccess: (saved) => {
        setSettings(saved);
        hapticFeedback('success');
      },
      onError: () => hapticFeedback('error'),
    });
  }

  return (
    <Panel className="stack" aria-label="Настройки нотификаций">
      <div
        style={{
          fontSize: 'var(--text-md)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
        }}
      >
        🔔 Нотификации
      </div>

      <div style={rowStyle}>
        <span>Ежедневная сводка</span>
        <div className="row" style={{ gap: 'var(--space-3)' }}>
          <select
            aria-label="Час ежедневной сводки"
            value={settings.dailyHour}
            disabled={!settings.daily}
            onChange={(e) => patch('dailyHour', Number(e.target.value))}
            style={{
              background: 'var(--tg-secondary-bg)',
              border: 'none',
              borderRadius: 'var(--radius-sm)',
              padding: '4px 8px',
              fontSize: 'var(--text-sm)',
              opacity: settings.daily ? 1 : 0.5,
            }}
          >
            {HOURS.map((h) => (
              <option key={h} value={h}>
                {formatHour(h)}
              </option>
            ))}
          </select>
          <Toggle
            label="Ежедневная сводка"
            checked={settings.daily}
            onChange={(v) => onToggle('daily', v)}
          />
        </div>
      </div>

      <div style={rowStyle}>
        <span>Напоминание о стрике</span>
        <Toggle
          label="Напоминание о стрике"
          checked={settings.streakReminder}
          onChange={(v) => onToggle('streakReminder', v)}
        />
      </div>

      <div style={rowStyle}>
        <span>Утренние квесты</span>
        <Toggle
          label="Утренние квесты"
          checked={settings.morning}
          onChange={(v) => onToggle('morning', v)}
        />
      </div>

      <div style={rowStyle}>
        <span>Вехи (уровень / ранг)</span>
        <Toggle
          label="Вехи"
          checked={settings.milestone}
          onChange={(v) => onToggle('milestone', v)}
        />
      </div>

      <div style={rowStyle}>
        <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
          Часовой пояс
        </span>
        <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
          {settings.timezone}
        </span>
      </div>

      {mutation.isError && (
        <div
          role="alert"
          style={{ color: 'var(--tg-destructive-text)', fontSize: 'var(--text-sm)' }}
        >
          Не удалось сохранить: {mutation.error.message}
        </div>
      )}
      {mutation.isSuccess && !mutation.isPending && (
        <div className="muted" style={{ fontSize: 'var(--text-sm)' }}>
          Сохранено ✓
        </div>
      )}

      <Button fullWidth onClick={onSave} disabled={mutation.isPending}>
        {mutation.isPending ? 'Сохранение…' : 'Сохранить'}
      </Button>
    </Panel>
  );
}
