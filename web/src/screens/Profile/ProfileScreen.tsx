/**
 * ProfileScreen — '/profile' hall of fame (docs/05 §5, docs/06 §6).
 *
 * Sections:
 *  - Header: name · class · rank.
 *  - 📊 Статистика — derived from GET /me + GET /report/today (the API exposes
 *    no dedicated lifetime-stats endpoint; we surface the metrics it does give:
 *    current streak, level, today's check-ins / XP).
 *  - 🏆 Ачивки — grid from useAchievements; locked ones rendered greyed out.
 *  - 🎒 Инвентарь — preview from useInventory.
 *  - 🔔 Нотификации — the settings block (PUT /settings/notifications).
 *
 * Loading / error / empty states per section.
 */

import type { CSSProperties, ReactNode } from 'react';
import { Panel, ProgressBar } from '../../components/ui';
import { useAchievements, useInventory, useMe, useTodayReport } from '../../lib/query';
import type {
  Achievement,
  CharacterClass,
  InventoryItem,
  Rank,
} from '../../types/api';
import NotificationSettings from '../Settings/NotificationSettings';
import { formatGold, itemIcon, RARITY_COLOR } from '../Settings/itemMeta';
import { EmptyState, ErrorState, LoadingState } from '../Settings/states';

// ── Labels ──────────────────────────────────────────────────────────────────

const CLASS_LABEL: Record<CharacterClass, string> = {
  warrior: 'Воин',
  sage: 'Мудрец',
  paladin: 'Паладин',
  druid: 'Друид',
  bard: 'Бард',
  adventurer: 'Авантюрист',
};

const RANK_META: Record<Rank, { icon: string; label: string }> = {
  recruit: { icon: '🎖️', label: 'Рекрут' },
  seeker: { icon: '🥈', label: 'Искатель' },
  veteran: { icon: '🥇', label: 'Ветеран' },
  master: { icon: '🏅', label: 'Мастер' },
  legend: { icon: '👑', label: 'Легенда' },
};

// ── Building blocks ───────────────────────────────────────────────────────────

function StatCell({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div
      style={{
        background: 'var(--tg-secondary-bg)',
        borderRadius: 'var(--radius-md)',
        padding: 'var(--space-3)',
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      <span
        style={{
          fontSize: 'var(--text-lg)',
          fontWeight: 'var(--weight-bold)' as unknown as number,
        }}
      >
        {value}
      </span>
      <span className="muted" style={{ fontSize: 'var(--text-xs)' }}>
        {label}
      </span>
    </div>
  );
}

const sectionTitle: CSSProperties = {
  fontSize: 'var(--text-md)',
  fontWeight: 'var(--weight-bold)' as unknown as number,
};

// ── Statistics section ──────────────────────────────────────────────────────

function StatisticsSection() {
  const me = useMe();
  const report = useTodayReport();

  if (me.isPending) return <LoadingState />;
  if (me.isError) return <ErrorState error={me.error} onRetry={() => void me.refetch()} />;

  const { character } = me.data;
  const checkinsToday = report.data?.checkins ?? me.data.todayCheckins.length;
  const xpToday = report.data?.xpGained ?? 0;
  const max = character.xpIntoLevel + character.xpToNext;

  return (
    <Panel className="stack" style={{ gap: 'var(--space-3)' }}>
      <div style={sectionTitle}>📊 Статистика</div>

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          gap: 'var(--space-2)',
        }}
      >
        <StatCell label="Стрик" value={`🔥 ${character.streakDays}`} />
        <StatCell label="Множитель стрика" value={`×${character.streakMult}`} />
        <StatCell label="Чек-инов сегодня" value={checkinsToday} />
        <StatCell label="XP сегодня" value={`+${xpToday}`} />
        <StatCell label="Уровень" value={character.level} />
        <StatCell label="Золото" value={`💰 ${formatGold(character.gold)}`} />
      </div>

      <div className="stack" style={{ gap: 'var(--space-1)' }}>
        <div className="row-between">
          <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
            Прогресс уровня
          </span>
          <span className="muted" style={{ fontSize: 'var(--text-xs)' }}>
            {character.xpIntoLevel} / {max} XP
          </span>
        </div>
        <ProgressBar value={character.xpIntoLevel} max={max} height={10} />
      </div>
    </Panel>
  );
}

// ── Achievements section ──────────────────────────────────────────────────────

function AchievementCell({ ach }: { ach: Achievement }) {
  const locked = !ach.unlocked;
  return (
    <div
      title={ach.description ?? ach.title}
      aria-label={`${ach.title}${locked ? ' (закрыто)' : ''}`}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 2,
        padding: 'var(--space-2)',
        background: 'var(--tg-secondary-bg)',
        borderRadius: 'var(--radius-md)',
        opacity: locked ? 0.4 : 1,
        filter: locked ? 'grayscale(1)' : 'none',
        textAlign: 'center',
      }}
    >
      <span aria-hidden style={{ fontSize: 'var(--text-xl)' }}>
        {locked ? '🔒' : ach.icon || '🏆'}
      </span>
      <span
        style={{
          fontSize: 'var(--text-xs)',
          color: locked ? 'var(--tg-hint)' : 'var(--tg-text)',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          maxWidth: '100%',
        }}
      >
        {ach.title}
      </span>
    </div>
  );
}

function AchievementsSection() {
  const { data, isPending, isError, error, refetch } = useAchievements();

  if (isPending) return <LoadingState />;
  if (isError) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const unlocked = data.filter((a) => a.unlocked).length;

  return (
    <Panel className="stack" style={{ gap: 'var(--space-3)' }}>
      <div style={sectionTitle}>
        🏆 Ачивки ({unlocked}/{data.length})
      </div>
      {data.length === 0 ? (
        <EmptyState icon="🏆" title="Пока нет ачивок" />
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(64px, 1fr))',
            gap: 'var(--space-2)',
          }}
        >
          {data.map((a) => (
            <AchievementCell key={a.id} ach={a} />
          ))}
        </div>
      )}
    </Panel>
  );
}

// ── Inventory preview section ─────────────────────────────────────────────────

function InventoryPreviewItem({ item }: { item: InventoryItem }) {
  return (
    <div
      title={item.name}
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 2,
        padding: 'var(--space-2)',
        background: 'var(--tg-secondary-bg)',
        borderRadius: 'var(--radius-md)',
        border: item.equipped ? `2px solid ${RARITY_COLOR[item.rarity]}` : '2px solid transparent',
        minWidth: 0,
      }}
    >
      <span aria-hidden style={{ fontSize: 'var(--text-xl)' }}>
        {itemIcon(item.slot, item.icon)}
      </span>
      <span
        style={{
          fontSize: 'var(--text-xs)',
          color: RARITY_COLOR[item.rarity],
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          maxWidth: '100%',
        }}
      >
        {item.name}
      </span>
    </div>
  );
}

function InventoryPreviewSection() {
  const { data, isPending, isError, error, refetch } = useInventory();

  if (isPending) return <LoadingState />;
  if (isError) return <ErrorState error={error} onRetry={() => void refetch()} />;

  const preview = data.slice(0, 6);

  return (
    <Panel className="stack" style={{ gap: 'var(--space-3)' }}>
      <div style={sectionTitle}>🎒 Инвентарь</div>
      {data.length === 0 ? (
        <EmptyState icon="🎒" title="Инвентарь пуст" hint="Загляни в магазин." />
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(72px, 1fr))',
            gap: 'var(--space-2)',
          }}
        >
          {preview.map((it) => (
            <InventoryPreviewItem key={it.id} item={it} />
          ))}
        </div>
      )}
    </Panel>
  );
}

// ── Header ────────────────────────────────────────────────────────────────────

function ProfileHeader() {
  const me = useMe();
  if (!me.data) return null;
  const { character } = me.data;
  const rank = RANK_META[character.rank];

  return (
    <div className="stack" style={{ gap: 2 }}>
      <h1 style={{ fontSize: 'var(--text-xl)' }}>👤 {character.name}</h1>
      <div className="muted" style={{ fontSize: 'var(--text-sm)' }}>
        {CLASS_LABEL[character.class]} · {rank.icon} {rank.label}
      </div>
    </div>
  );
}

// ── Screen ──────────────────────────────────────────────────────────────────

export default function ProfileScreen() {
  return (
    <div className="stack" style={{ gap: 'var(--space-4)' }}>
      <ProfileHeader />
      <StatisticsSection />
      <AchievementsSection />
      <InventoryPreviewSection />
      <NotificationSettings />
    </div>
  );
}
