/**
 * RewardOverlay — the "juice" layer (docs/05 §6, docs/12 §10).
 *
 * Drains the reward-event queue from the store one event at a time and plays it
 * as a sequence of staged scenes, composed from RewardPieces:
 *
 *   stage 0 → +XP / +Gold badge (+ crit flash if isCrit) + streak chip
 *   stage 1 → level-up banner (confetti, pulsing number)   [if levelUp]
 *   stage 2 → rank-up ceremonial overlay                    [if rankUp]
 *   stage 3 → item drop card by rarity                      [if drop]
 *
 * Behaviour:
 *  - Optimistic: events appear instantly when enqueued by the check-in flow.
 *  - Telegram Haptic fires on crit / level-up / rank-up (docs/12 §10).
 *  - Tap anywhere advances to the next stage; after the last stage the event is
 *    dequeued and the next one (if any) starts.
 *  - prefers-reduced-motion: timings collapse, particles suppressed, auto-advance
 *    is faster (handled by sub-pieces + this layer).
 *
 * Mounted once at the app root region is not required — it self-manages via the
 * store, so it can live wherever it is rendered (we render it from HeroScreen so
 * it overlays the whole viewport via fixed positioning).
 */

import { AnimatePresence, motion } from 'framer-motion';
import { useCallback, useEffect, useMemo, useState } from 'react';
import type { CSSProperties } from 'react';
import type { RewardEvent } from '../../types/api';
import { useStore } from '../../store/useStore';
import { hapticFeedback } from '../../telegram/sdk';
import {
  CritFlash,
  DropCard,
  LevelUpBanner,
  RankUpOverlay,
  StreakChip,
  XpGoldBadge,
} from './RewardPieces';
import { rankFromLevel } from './labels';
import { useReducedMotion } from './useReducedMotion';

type Stage = 'reward' | 'levelUp' | 'rankUp' | 'drop';

/** Build the ordered list of stages this event needs. */
function stagesFor(event: RewardEvent): Stage[] {
  const stages: Stage[] = ['reward'];
  if (event.levelUp) stages.push('levelUp');
  if (event.rankUp) stages.push('rankUp');
  if (event.drop) stages.push('drop');
  return stages;
}

/** Auto-advance timing per stage (ms). Reduced motion shortens everything. */
function stageDuration(stage: Stage, reduced: boolean): number {
  if (reduced) return 900;
  switch (stage) {
    case 'reward':
      return 1600;
    case 'levelUp':
      return 2400;
    case 'rankUp':
      return 2800;
    case 'drop':
      return 2600;
  }
}

const overlayStyle: CSSProperties = {
  position: 'fixed',
  inset: 0,
  zIndex: 'var(--z-reward)' as unknown as number,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  padding: 'var(--space-6)',
  background: 'rgba(0, 0, 0, 0.45)',
  cursor: 'pointer',
};

export function RewardOverlay() {
  const reduced = useReducedMotion();
  const queue = useStore((s) => s.rewardQueue);
  const dequeueReward = useStore((s) => s.dequeueReward);

  const current: RewardEvent | undefined = queue[0];
  const [stageIndex, setStageIndex] = useState(0);

  const stages = useMemo<Stage[]>(
    () => (current ? stagesFor(current) : []),
    [current],
  );
  const stage: Stage | undefined = stages[stageIndex];

  // Reset stage pointer whenever a new event reaches the head of the queue.
  useEffect(() => {
    setStageIndex(0);
  }, [current]);

  // Haptics on entering a punchy stage (docs/12 §10).
  useEffect(() => {
    if (!current || !stage) return;
    if (stage === 'reward' && current.reward.isCrit) {
      hapticFeedback('heavy');
    } else if (stage === 'levelUp') {
      hapticFeedback('success');
    } else if (stage === 'rankUp') {
      hapticFeedback('success');
    } else if (stage === 'drop') {
      hapticFeedback('rigid');
    }
  }, [current, stage]);

  const advance = useCallback(() => {
    if (!current) return;
    if (stageIndex + 1 < stages.length) {
      setStageIndex((i) => i + 1);
    } else {
      // Last stage done → drop this event; queue effect picks up the next.
      dequeueReward();
    }
  }, [current, stageIndex, stages.length, dequeueReward]);

  // Auto-advance timer for the active stage.
  useEffect(() => {
    if (!current || !stage) return;
    const ms = stageDuration(stage, reduced);
    const id = window.setTimeout(advance, ms);
    return () => window.clearTimeout(id);
  }, [current, stage, reduced, advance]);

  // Crit doubles the roll (docs/01); the reward payload has no explicit factor.
  const critMultiplier = 2;

  return (
    <AnimatePresence>
      {current && stage && (
        <motion.div
          key="reward-overlay"
          style={overlayStyle}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: reduced ? 0.001 : 0.18 }}
          onClick={advance}
          role="presentation"
          aria-live="polite"
        >
          <div
            style={{
              position: 'relative',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 'var(--space-4)',
              width: '100%',
              maxWidth: 'var(--content-max-width)',
            }}
          >
            <AnimatePresence mode="wait">
              {stage === 'reward' && (
                <motion.div
                  key="stage-reward"
                  style={{ display: 'contents' }}
                  exit={{ opacity: 0 }}
                >
                  {current.reward.isCrit && <CritFlash multiplier={critMultiplier} />}
                  <XpGoldBadge reward={current.reward} />
                  <StreakChip
                    days={current.reward.streakDays}
                    mult={current.reward.streakMult}
                  />
                </motion.div>
              )}

              {stage === 'levelUp' && current.levelUp && (
                <motion.div key="stage-level" exit={{ opacity: 0 }}>
                  <LevelUpBanner transition={current.levelUp} />
                </motion.div>
              )}

              {stage === 'rankUp' && current.rankUp && (
                <motion.div key="stage-rank" exit={{ opacity: 0 }}>
                  <RankUpOverlay
                    fromRank={rankFromLevel(current.rankUp.from)}
                    toRank={rankFromLevel(current.rankUp.to)}
                  />
                </motion.div>
              )}

              {stage === 'drop' && current.drop && (
                <motion.div key="stage-drop" exit={{ opacity: 0 }}>
                  <DropCard drop={current.drop} />
                </motion.div>
              )}
            </AnimatePresence>

            <div
              style={{
                fontSize: 'var(--text-xs)',
                color: 'rgba(255,255,255,0.7)',
                marginTop: 'var(--space-2)',
              }}
            >
              тапни, чтобы продолжить
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

export default RewardOverlay;
