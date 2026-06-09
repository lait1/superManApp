/**
 * OnboardingScreen — first-run flow: RPG welcome + character creation.
 *
 * Rendered by App INSTEAD of the routed shell while `me.character.onboarded`
 * is false. Two steps:
 *
 *   1. welcome — "your life is an RPG" pitch;
 *   2. creator — live CharacterCanvas preview + appearance pickers
 *      (body type / skin tone / hairstyle / hair color) + hero name.
 *
 * Submitting POSTs /character/setup; useSetupCharacter patches the cached
 * /me, the gate in App opens and the user lands on the Hero screen.
 */

import { useMemo, useState, type CSSProperties } from 'react';
import { Button, Panel } from '../../components/ui';
import { CharacterCanvas } from '../../character/CharacterCanvas';
import { useCharacterAssets } from '../../character/useCharacterAssets';
import { useMe, useSetupCharacter } from '../../lib/query';
import { hapticFeedback } from '../../telegram/sdk';
import type { CharacterAppearance } from '../../types/api';

// Swatch colors mirror the base tone of each generator ramp (assets/palette.json).
const SKIN_SWATCHES: Record<string, string> = {
  s1: '#E3B189',
  s2: '#B07A52',
  s3: '#8A5638',
  s4: '#5F3B2A',
};

const HAIR_SWATCHES: Record<string, string> = {
  dark: '#433D4D',
  brown: '#8A5A2E',
  blond: '#E8C465',
  red: '#D6562C',
};

const BODY_TYPE_LABELS: Record<string, string> = {
  a: 'Крепкое',
  b: 'Стройное',
};

const HAIRSTYLE_LABELS: Record<string, string> = {
  bald: 'Без волос',
  short: 'Короткая',
  spiky: 'Ёжик',
  long: 'Длинная',
  ponytail: 'Хвост',
};

const MAX_NAME_LEN = 24;

/** Placeholder name assigned by the backend before onboarding. */
const DEFAULT_BACKEND_NAME = 'superMen';

export default function OnboardingScreen() {
  const [step, setStep] = useState<'welcome' | 'creator'>('welcome');

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 'var(--space-4)',
        padding: 'var(--space-4)',
        maxWidth: 480,
        margin: '0 auto',
      }}
    >
      {step === 'welcome' ? (
        <WelcomeStep onNext={() => setStep('creator')} />
      ) : (
        <CreatorStep />
      )}
    </div>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Step 1 — welcome
// ──────────────────────────────────────────────────────────────────────────

function WelcomeStep({ onNext }: { onNext: () => void }) {
  const rows: Array<[string, string]> = [
    ['⚔️', 'Реальные дела — спорт, языки, работа — это твои квесты.'],
    ['📈', 'Отмечай их чек-инами: герой получает XP, золото и качает статы.'],
    ['🛡️', 'Расти в рангах, открывай классы, собирай экипировку.'],
  ];

  return (
    <>
      <h1 style={{ textAlign: 'center', margin: 'var(--space-4) 0 0' }}>
        Твоя жизнь — это RPG
      </h1>
      <p className="muted" style={{ textAlign: 'center', margin: 0 }}>
        Прокачивай себя — и твой герой прокачается вместе с тобой.
      </p>
      <Panel>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-3)' }}>
          {rows.map(([icon, text]) => (
            <div key={icon} style={{ display: 'flex', gap: 'var(--space-3)', alignItems: 'center' }}>
              <span style={{ fontSize: 24 }}>{icon}</span>
              <span>{text}</span>
            </div>
          ))}
        </div>
      </Panel>
      <Button size="lg" fullWidth onClick={onNext}>
        Создать героя
      </Button>
    </>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Step 2 — character creator
// ──────────────────────────────────────────────────────────────────────────

function CreatorStep() {
  const { data: manifest } = useCharacterAssets();
  const { data: me } = useMe();
  const setup = useSetupCharacter();

  const initialName =
    me && me.character.name !== DEFAULT_BACKEND_NAME ? me.character.name : '';
  const [name, setName] = useState(initialName);

  const defaults: CharacterAppearance = useMemo(
    () =>
      manifest?.defaults ?? {
        bodyType: 'a',
        skinTone: 's2',
        hairstyle: 'short',
        hairColor: 'dark',
      },
    [manifest],
  );
  const [appearance, setAppearance] = useState<CharacterAppearance | null>(null);
  const look = appearance ?? defaults;

  const patch = (field: keyof CharacterAppearance, value: string) => {
    hapticFeedback('selection');
    setAppearance({ ...look, [field]: value });
  };

  const trimmedName = name.trim();
  const canSubmit = trimmedName.length > 0 && trimmedName.length <= MAX_NAME_LEN && !setup.isPending;

  const submit = () => {
    if (!canSubmit) return;
    hapticFeedback('medium');
    setup.mutate({ name: trimmedName, appearance: look });
  };

  return (
    <>
      <h2 style={{ textAlign: 'center', margin: 'var(--space-2) 0 0' }}>Твой герой</h2>

      <Panel style={{ display: 'flex', justifyContent: 'center' }}>
        <CharacterCanvas
          scale={2}
          overrides={{ class: 'adventurer', stage: 1, appearance: look }}
          ariaLabel="Предпросмотр героя"
        />
      </Panel>

      <Panel style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-4)' }}>
        <Field label="Имя героя">
          <input
            value={name}
            maxLength={MAX_NAME_LEN}
            placeholder="Например, Странник"
            onChange={(e) => setName(e.target.value)}
            style={inputStyle}
          />
        </Field>

        <Field label="Телосложение">
          <ChipRow
            options={manifest?.bodyTypes ?? Object.keys(BODY_TYPE_LABELS)}
            labels={BODY_TYPE_LABELS}
            value={look.bodyType}
            onSelect={(v) => patch('bodyType', v)}
          />
        </Field>

        <Field label="Тон кожи">
          <SwatchRow
            options={manifest?.skinTones ?? Object.keys(SKIN_SWATCHES)}
            colors={SKIN_SWATCHES}
            value={look.skinTone}
            onSelect={(v) => patch('skinTone', v)}
          />
        </Field>

        <Field label="Причёска">
          <ChipRow
            options={manifest?.hairstyles ?? Object.keys(HAIRSTYLE_LABELS)}
            labels={HAIRSTYLE_LABELS}
            value={look.hairstyle}
            onSelect={(v) => patch('hairstyle', v)}
          />
        </Field>

        {look.hairstyle !== 'bald' && (
          <Field label="Цвет волос">
            <SwatchRow
              options={manifest?.hairColors ?? Object.keys(HAIR_SWATCHES)}
              colors={HAIR_SWATCHES}
              value={look.hairColor}
              onSelect={(v) => patch('hairColor', v)}
            />
          </Field>
        )}
      </Panel>

      {setup.isError && (
        <p style={{ color: 'var(--tg-destructive-text)', textAlign: 'center', margin: 0 }}>
          Не получилось сохранить героя: {setup.error.message}
        </p>
      )}

      <Button size="lg" fullWidth disabled={!canSubmit} onClick={submit}>
        {setup.isPending ? 'Создаём…' : 'В путь!'}
      </Button>
    </>
  );
}

// ──────────────────────────────────────────────────────────────────────────
// Small picker primitives
// ──────────────────────────────────────────────────────────────────────────

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-2)' }}>
      <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
        {label}
      </span>
      {children}
    </div>
  );
}

const inputStyle: CSSProperties = {
  background: 'var(--tg-secondary-bg)',
  color: 'var(--tg-text)',
  border: '1px solid var(--scene-border)',
  borderRadius: 'var(--radius-md, 8px)',
  padding: '10px 12px',
  fontSize: 'var(--text-md, 15px)',
  outline: 'none',
  width: '100%',
  boxSizing: 'border-box',
};

function ChipRow({
  options,
  labels,
  value,
  onSelect,
}: {
  options: string[];
  labels: Record<string, string>;
  value: string;
  onSelect: (v: string) => void;
}) {
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--space-2)' }}>
      {options.map((opt) => {
        const active = opt === value;
        return (
          <button
            key={opt}
            type="button"
            onClick={() => onSelect(opt)}
            aria-pressed={active}
            style={{
              padding: '8px 14px',
              borderRadius: 999,
              border: active
                ? '1px solid var(--tg-button)'
                : '1px solid var(--scene-border)',
              background: active ? 'var(--tg-button)' : 'var(--tg-secondary-bg)',
              color: active ? 'var(--tg-button-text)' : 'var(--tg-text)',
              fontSize: 'var(--text-sm)',
              cursor: 'pointer',
            }}
          >
            {labels[opt] ?? opt}
          </button>
        );
      })}
    </div>
  );
}

function SwatchRow({
  options,
  colors,
  value,
  onSelect,
}: {
  options: string[];
  colors: Record<string, string>;
  value: string;
  onSelect: (v: string) => void;
}) {
  return (
    <div style={{ display: 'flex', gap: 'var(--space-3)' }}>
      {options.map((opt) => {
        const active = opt === value;
        return (
          <button
            key={opt}
            type="button"
            onClick={() => onSelect(opt)}
            aria-pressed={active}
            aria-label={opt}
            style={{
              width: 36,
              height: 36,
              borderRadius: '50%',
              border: active ? '3px solid var(--tg-button)' : '2px solid var(--scene-border)',
              background: colors[opt] ?? '#888',
              cursor: 'pointer',
            }}
          />
        );
      })}
    </div>
  );
}
