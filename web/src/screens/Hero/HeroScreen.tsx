/**
 * HeroScreen — main screen ('/'), docs/05 §2. The face of the app and the core
 * of the loop.
 *
 * Layout (top → bottom), matching the wireframe:
 *   rank · gold · streak           (header strip)
 *   name · class · level           + XP progress bar
 *   <CharacterCanvas/>             inside a PixelFrame scene panel
 *   5 stat rows (StatBar)
 *   today's check-ins
 *   "Отметить" FAB-style button    (opens the check-in modal via the store)
 *
 * Data comes from useMe(); the latest character snapshot is mirrored into the
 * zustand store so the TabBar / overlay can read it synchronously. The reward
 * overlay (docs/05 §6) is mounted here so it can paint over the whole viewport.
 */

import { useEffect } from 'react';
import type { CSSProperties } from 'react';
import { Button, Panel, PixelFrame, ProgressBar, StatBar } from '../../components/ui';
import { CharacterCanvas } from '../../character/CharacterCanvas';
import { RewardOverlay } from '../../components/reward';
import { CLASS_META, RANK_META } from '../../components/reward/labels';
import { useMe } from '../../lib/query';
import { useStore } from '../../store/useStore';
import type { StatKey } from '../../types/api';

/** Fixed stat order for the rows (docs/05 §2). */
const STAT_ORDER: readonly StatKey[] = ['STR', 'INT', 'DIS', 'VIT', 'CHA'];

const headerStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 'var(--space-3)',
};

function nf(n: number): string {
  // Grouped thousands ("6 420") to match the wireframe.
  return n.toLocaleString('ru-RU');
}

export default function HeroScreen() {
  const { data, isLoading, isError, error, refetch } = useMe();
  const setCharacter = useStore((s) => s.setCharacter);
  const openCheckin = useStore((s) => s.openCheckin);

  // Mirror the freshest character snapshot into the store for the TabBar/overlay.
  useEffect(() => {
    if (data) setCharacter(data.character);
  }, [data, setCharacter]);

  if (isLoading) {
    return (
      <>
        <div className="muted" style={{ padding: 'var(--space-6)', textAlign: 'center' }}>
          Загрузка героя…
        </div>
        <RewardOverlay />
      </>
    );
  }

  if (isError || !data) {
    return (
      <div className="stack" style={{ padding: 'var(--space-6)', textAlign: 'center' }}>
        <p>Не удалось загрузить героя.</p>
        <p className="muted" style={{ fontSize: 'var(--text-sm)' }}>
          {error?.message ?? 'Попробуй ещё раз.'}
        </p>
        <div>
          <Button onClick={() => void refetch()}>Повторить</Button>
        </div>
      </div>
    );
  }

  const { character, stats, todayCheckins } = data;
  const cls = CLASS_META[character.class];
  const rank = RANK_META[character.rank];
  const statByKey = new Map(stats.map((s) => [s.key, s]));

  return (
    <div className="stack" style={{ gap: 'var(--space-4)' }}>
      {/* ── header: rank · gold · streak ─────────────────────────────────── */}
      <div style={headerStyle}>
        <span
          className="row"
          style={{ gap: 'var(--space-1)', color: rank.color, fontWeight: 700 }}
        >
          <span aria-hidden>{rank.icon}</span>
          {rank.label}
        </span>
        <span className="row" style={{ gap: 'var(--space-4)', fontSize: 'var(--text-md)' }}>
          <span className="row" style={{ gap: 4 }}>
            <span aria-hidden>💰</span>
            {nf(character.gold)}
          </span>
          <span
            className="row"
            style={{ gap: 4, color: 'var(--class-warrior)', fontWeight: 700 }}
          >
            <span aria-hidden>🔥</span>
            {character.streakDays}
          </span>
        </span>
      </div>

      {/* ── name · class · level + XP bar ────────────────────────────────── */}
      <div className="stack" style={{ gap: 'var(--space-2)' }}>
        <div style={headerStyle}>
          <span
            className="row"
            style={{ gap: 'var(--space-2)', fontSize: 'var(--text-lg)', fontWeight: 700 }}
          >
            <span aria-hidden>{cls.icon}</span>
            <span>{character.name}</span>
            <span className="muted" style={{ fontWeight: 400 }}>
              · {cls.label}
            </span>
          </span>
          <span className="text-pixel" style={{ fontSize: 'var(--text-md)' }}>
            LVL {character.level}
          </span>
        </div>
        <ProgressBar
          value={character.xpIntoLevel}
          max={character.xpIntoLevel + character.xpToNext}
          color={cls.color}
          height={12}
        />
        <div
          style={{
            fontSize: 'var(--text-xs)',
            color: 'var(--tg-hint)',
            textAlign: 'right',
          }}
        >
          {nf(character.xpIntoLevel)} / {nf(character.xpIntoLevel + character.xpToNext)} XP
        </div>
      </div>

      {/* ── character ────────────────────────────────────────────────────── */}
      <PixelFrame style={{ padding: 'var(--space-4)', alignSelf: 'center', width: '100%' }}>
        <CharacterCanvas
          scale={3}
          characterClass={character.class}
          rank={character.rank}
        />
      </PixelFrame>

      {/* ── stats ────────────────────────────────────────────────────────── */}
      <Panel className="stack" style={{ gap: 'var(--space-3)' }}>
        {STAT_ORDER.map((key) => {
          const s = statByKey.get(key);
          if (!s) return null;
          return (
            <StatBar
              key={key}
              statKey={key}
              intoLevel={s.intoLevel}
              toNext={s.toNext}
              level={s.level}
            />
          );
        })}
      </Panel>

      {/* ── today's check-ins ────────────────────────────────────────────── */}
      <Panel>
        <div
          style={{
            fontSize: 'var(--text-sm)',
            color: 'var(--tg-section-header-text)',
            marginBottom: 'var(--space-2)',
          }}
        >
          Сегодня
        </div>
        {todayCheckins.length === 0 ? (
          <p className="muted" style={{ fontSize: 'var(--text-sm)' }}>
            Отметь первое дело 💪
          </p>
        ) : (
          <div className="row" style={{ flexWrap: 'wrap', gap: 'var(--space-2)' }}>
            {todayCheckins.map((key) => (
              <span
                key={key}
                className="row"
                style={{
                  gap: 4,
                  padding: '4px 10px',
                  borderRadius: 'var(--radius-pill)',
                  background: 'var(--tg-secondary-bg)',
                  fontSize: 'var(--text-sm)',
                }}
              >
                <span aria-hidden>✅</span>
                {key}
              </span>
            ))}
          </div>
        )}
      </Panel>

      {/* ── primary CTA (mirrors the FAB) ────────────────────────────────── */}
      <Button size="lg" fullWidth onClick={openCheckin} style={{ marginTop: 'var(--space-2)' }}>
        ⊕ Отметить
      </Button>

      {/* Reward "juice" layer — paints over the viewport when events arrive. */}
      <RewardOverlay />
    </div>
  );
}
