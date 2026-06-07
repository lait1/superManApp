/**
 * CheckinModal — the check-in flow (docs/05 §3), rendered above all routes.
 *
 * Goal: 1–2 taps to log an activity. Open/close state lives in the zustand store
 * (the FAB in TabBar opens it). Flow:
 *
 *   1. pick an activity from the grid (useActivities catalog)
 *   2. (optional) duration if the activity hasDuration; (optional) note
 *   3. tap "Отметить +XP" → POST /checkin (useCheckin mutation)
 *
 * Optimistic UX: the modal closes immediately on submit and the resulting
 * RewardEvent is pushed into the store's reward queue, which the RewardOverlay
 * (docs/05 §6) drains and animates. React Query invalidation in useCheckin then
 * refreshes /me, /quests, /achievements in the background.
 */

import { useEffect, useMemo, useState } from 'react';
import type { CSSProperties } from 'react';
import { Button, Modal } from '../../components/ui';
import { RARITY_META } from '../../components/reward/labels';
import { useActivities, useCheckin } from '../../lib/query';
import { useStore } from '../../store/useStore';
import { hapticFeedback } from '../../telegram/sdk';
import type { Activity, StatKey } from '../../types/api';

/** Fallback icons per stat when the activity has no icon (docs/05 §3 grid). */
const STAT_ICON: Record<StatKey, string> = {
  STR: '🏋️',
  INT: '🧠',
  DIS: '🎯',
  VIT: '🧘',
  CHA: '🤝',
};

const gridStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns: 'repeat(3, 1fr)',
  gap: 'var(--space-2)',
};

function tileStyle(selected: boolean, accent: string): CSSProperties {
  return {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 'var(--space-1)',
    padding: 'var(--space-3)',
    minHeight: 76,
    borderRadius: 'var(--radius-md)',
    background: selected ? 'var(--tg-secondary-bg)' : 'var(--tg-section-bg)',
    border: `2px solid ${selected ? accent : 'transparent'}`,
    color: 'var(--tg-text)',
    textAlign: 'center',
    transition:
      'border-color var(--dur-fast) var(--ease-standard), background var(--dur-fast) var(--ease-standard)',
  };
}

const inputStyle: CSSProperties = {
  width: '100%',
  padding: '10px 12px',
  borderRadius: 'var(--radius-md)',
  background: 'var(--tg-secondary-bg)',
  border: '1px solid var(--tg-secondary-bg)',
  outline: 'none',
};

export default function CheckinModal() {
  const open = useStore((s) => s.checkinOpen);
  const closeCheckin = useStore((s) => s.closeCheckin);
  const enqueueReward = useStore((s) => s.enqueueReward);

  const { data: activities, isLoading, isError } = useActivities();
  const checkin = useCheckin();

  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [duration, setDuration] = useState<string>('');
  const [note, setNote] = useState<string>('');

  // Reset the form each time the modal opens.
  useEffect(() => {
    if (open) {
      setSelectedKey(null);
      setDuration('');
      setNote('');
      checkin.reset();
    }
    // Only re-run when the open flag flips; `checkin` is a stable mutation obj.
  }, [open]);

  const selected: Activity | undefined = useMemo(
    () => activities?.find((a) => a.key === selectedKey),
    [activities, selectedKey],
  );

  const canSubmit = Boolean(selected) && !checkin.isPending;

  function handleSelect(activity: Activity): void {
    hapticFeedback('selection');
    setSelectedKey(activity.key);
  }

  function handleSubmit(): void {
    if (!selected) return;

    const trimmedNote = note.trim();
    const parsedDuration = Number.parseInt(duration, 10);
    const durationMin =
      selected.hasDuration && Number.isFinite(parsedDuration) && parsedDuration > 0
        ? parsedDuration
        : undefined;

    checkin.mutate(
      {
        activityKey: selected.key,
        ...(durationMin !== undefined ? { durationMin } : {}),
        ...(trimmedNote ? { note: trimmedNote } : {}),
      },
      {
        onSuccess: (event) => {
          enqueueReward(event);
        },
      },
    );

    // Optimistic: close immediately; the reward overlay plays once the event
    // is enqueued (network resolves moments later).
    hapticFeedback('light');
    closeCheckin();
  }

  return (
    <Modal open={open} onClose={closeCheckin} title="Что сделал?">
      {isLoading && <p className="muted">Загрузка активностей…</p>}

      {isError && (
        <p style={{ color: 'var(--tg-destructive-text)' }}>
          Не удалось загрузить активности. Попробуй позже.
        </p>
      )}

      {activities && activities.length > 0 && (
        <div className="stack" style={{ gap: 'var(--space-4)' }}>
          <div style={gridStyle}>
            {activities.map((a) => {
              const isSel = a.key === selectedKey;
              const accent = RARITY_META[a.rarity].color;
              return (
                <button
                  key={a.key}
                  type="button"
                  style={tileStyle(isSel, accent)}
                  aria-pressed={isSel}
                  onClick={() => handleSelect(a)}
                >
                  <span aria-hidden style={{ fontSize: 'var(--text-xl)' }}>
                    {a.icon ?? STAT_ICON[a.statKey]}
                  </span>
                  <span style={{ fontSize: 'var(--text-xs)' }}>{a.title}</span>
                </button>
              );
            })}
          </div>

          {selected?.hasDuration && (
            <label className="stack" style={{ gap: 'var(--space-1)' }}>
              <span style={{ fontSize: 'var(--text-sm)', color: 'var(--tg-hint)' }}>
                ⏱️ Длительность (мин, опц.)
              </span>
              <input
                style={inputStyle}
                type="number"
                inputMode="numeric"
                min={0}
                placeholder={
                  selected.refMinutes ? String(selected.refMinutes) : '45'
                }
                value={duration}
                onChange={(e) => setDuration(e.target.value)}
              />
            </label>
          )}

          <label className="stack" style={{ gap: 'var(--space-1)' }}>
            <span style={{ fontSize: 'var(--text-sm)', color: 'var(--tg-hint)' }}>
              📝 Заметка (опц.)
            </span>
            <input
              style={inputStyle}
              type="text"
              maxLength={140}
              placeholder="…"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>

          {checkin.isError && (
            <p style={{ color: 'var(--tg-destructive-text)', fontSize: 'var(--text-sm)' }}>
              {checkin.error.message}
            </p>
          )}

          <Button
            size="lg"
            fullWidth
            disabled={!canSubmit}
            onClick={handleSubmit}
          >
            {checkin.isPending ? 'Отмечаю…' : 'Отметить +XP'}
          </Button>
        </div>
      )}

      {activities && activities.length === 0 && (
        <p className="muted">Активности не найдены.</p>
      )}
    </Modal>
  );
}
