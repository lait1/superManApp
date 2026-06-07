/**
 * QuestsScreen — '/quests' (docs/05 §4, docs/02 §2).
 *
 * Tabs: Сегодня (daily) / Неделя (weekly) / Цепочки (chains). Each quest renders
 * a progress bar, its reward (XP / gold / optional title / item), and a «Забрать»
 * button when completed (POST /quests/{id}/claim via useClaimQuest). Handles
 * loading / error / empty states.
 */

import { useMemo, useState } from 'react';
import type { CSSProperties } from 'react';
import { Button, Panel, ProgressBar } from '../../components/ui';
import { useClaimQuest, useQuests } from '../../lib/query';
import { hapticFeedback } from '../../telegram/sdk';
import type { Quest } from '../../types/api';
import { formatGold } from '../Settings/itemMeta';
import { EmptyState, ErrorState, LoadingState } from '../Settings/states';
import { Tabs, type TabOption } from '../Settings/Tabs';

type QuestTab = 'daily' | 'weekly' | 'chains';

const TAB_OPTIONS: ReadonlyArray<TabOption<QuestTab>> = [
  { key: 'daily', label: 'Сегодня' },
  { key: 'weekly', label: 'Неделя' },
  { key: 'chains', label: 'Цепочки' },
];

const EMPTY_HINT: Record<QuestTab, string> = {
  daily: 'На сегодня квестов нет. Отметь дело — появятся новые.',
  weekly: 'Недельных квестов пока нет.',
  chains: 'Активных цепочек нет. Они открываются по мере прогресса.',
};

function RewardLine({ quest }: { quest: Quest }) {
  const { reward } = quest;
  return (
    <span
      className="muted"
      style={{ fontSize: 'var(--text-sm)', display: 'inline-flex', gap: 'var(--space-3)', flexWrap: 'wrap' }}
    >
      {reward.xp > 0 && <span style={{ color: 'var(--tg-link)' }}>+{reward.xp} XP</span>}
      {reward.gold > 0 && <span>💰 {formatGold(reward.gold)}</span>}
      {reward.title && <span>🏷️ «{reward.title}»</span>}
      {reward.item && <span>🎁 {reward.item}</span>}
    </span>
  );
}

const cardStyle: CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: 'var(--space-2)',
};

function statusMark(quest: Quest): string {
  switch (quest.status) {
    case 'claimed':
      return '☑';
    case 'completed':
      return '✅';
    case 'expired':
      return '⌛';
    case 'active':
    default:
      return '☐';
  }
}

function QuestCard({ quest }: { quest: Quest }) {
  const claim = useClaimQuest();
  const claimable = quest.status === 'completed';
  const claimed = quest.status === 'claimed';

  function onClaim(): void {
    claim.mutate(quest.id, {
      onSuccess: () => hapticFeedback('success'),
      onError: () => hapticFeedback('error'),
    });
  }

  return (
    <Panel style={cardStyle}>
      <div className="row-between" style={{ gap: 'var(--space-2)', alignItems: 'flex-start' }}>
        <div className="row" style={{ gap: 'var(--space-2)', alignItems: 'flex-start' }}>
          <span aria-hidden style={{ fontSize: 'var(--text-md)' }}>
            {statusMark(quest)}
          </span>
          <span style={{ fontWeight: 'var(--weight-medium)' as unknown as number }}>
            {quest.title}
          </span>
        </div>
        {claimed && (
          <span style={{ color: 'var(--rarity-uncommon)', fontSize: 'var(--text-sm)' }}>
            Получено
          </span>
        )}
        {quest.status === 'expired' && (
          <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
            Истёк
          </span>
        )}
      </div>

      <ProgressBar
        value={quest.progress}
        max={quest.target}
        color={claimable || claimed ? 'var(--rarity-uncommon)' : 'var(--tg-button)'}
        height={10}
      />
      <div className="row-between">
        <RewardLine quest={quest} />
        <span className="muted" style={{ fontSize: 'var(--text-xs)', whiteSpace: 'nowrap' }}>
          {quest.progress}/{quest.target}
        </span>
      </div>

      {claimable && (
        <>
          <Button
            fullWidth
            size="sm"
            onClick={onClaim}
            disabled={claim.isPending}
            style={{ marginTop: 'var(--space-1)' }}
          >
            {claim.isPending ? 'Забираем…' : 'Забрать награду'}
          </Button>
          {claim.isError && (
            <span
              role="alert"
              style={{ color: 'var(--tg-destructive-text)', fontSize: 'var(--text-sm)' }}
            >
              {claim.error.message}
            </span>
          )}
        </>
      )}
    </Panel>
  );
}

export default function QuestsScreen() {
  const [tab, setTab] = useState<QuestTab>('daily');
  const { data, isPending, isError, error, refetch } = useQuests();

  const quests = useMemo<Quest[]>(() => {
    if (!data) return [];
    return data[tab];
  }, [data, tab]);

  return (
    <div className="stack" style={{ gap: 'var(--space-4)' }}>
      <h1 style={{ fontSize: 'var(--text-xl)' }}>⚔️ Квесты</h1>
      <Tabs options={TAB_OPTIONS} value={tab} onChange={setTab} />

      {isPending && <LoadingState />}
      {isError && <ErrorState error={error} onRetry={() => void refetch()} />}

      {!isPending && !isError && (
        <div className="stack" style={{ gap: 'var(--space-3)' }}>
          {quests.length === 0 ? (
            <EmptyState icon="🗺️" title="Пусто" hint={EMPTY_HINT[tab]} />
          ) : (
            quests.map((q) => <QuestCard key={q.id} quest={q} />)
          )}
        </div>
      )}
    </div>
  );
}
