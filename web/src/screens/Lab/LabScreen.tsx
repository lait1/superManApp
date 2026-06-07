/**
 * LabScreen — Character Lab ('/lab', docs/12 §12 — consistency tool).
 *
 * An interactive harness over <CharacterCanvas/> for the quality council:
 *   • toggle class (6), rank stage (1–5)
 *   • equip/clear each item slot, picking from the manifest by rarity
 *   • a live preview of the chosen combination
 *   • a gallery that renders every class × stage so silhouette/palette
 *     consistency can be eyeballed at a glance (docs/12 §1, §8, §9)
 *
 * Everything is manifest-driven, so adding an item/class to genassets surfaces
 * here automatically with no code change (docs/12 §11).
 */

import { useMemo, useState } from 'react';
import { Button, Panel, PixelFrame } from '../../components/ui';
import {
  CharacterCanvas,
  type EquippedItemMap,
} from '../../character/CharacterCanvas';
import {
  itemsForSlot,
  useCharacterAssets,
  type CharacterManifest,
  type ManifestItem,
} from '../../character/useCharacterAssets';
import type { CharacterClass, EquipSlot, Rarity } from '../../types/api';

// Slots the paper-doll renders (docs/12 §5). Ordered for a stable picker UI.
const RENDERED_SLOTS: EquipSlot[] = ['head', 'armor', 'weapon', 'amulet', 'back', 'boots'];

const CLASS_LABELS: Record<CharacterClass, string> = {
  warrior: 'Воин',
  sage: 'Мудрец',
  paladin: 'Паладин',
  druid: 'Друид',
  bard: 'Бард',
  adventurer: 'Авантюрист',
};

const SLOT_LABELS: Record<EquipSlot, string> = {
  weapon: 'Оружие',
  armor: 'Броня',
  amulet: 'Амулет',
  head: 'Голова',
  boots: 'Обувь',
  back: 'Спина',
  aura: 'Аура',
  background: 'Фон',
  consumable: 'Расходник',
};

const RARITY_LABELS: Record<Rarity, string> = {
  common: 'обыч.',
  uncommon: 'необ.',
  rare: 'ред.',
  epic: 'эпик',
  legendary: 'легенд.',
};

const RARITY_COLOR: Record<Rarity, string> = {
  common: '#9aa4ad',
  uncommon: '#4caf6d',
  rare: '#3b82f6',
  epic: '#a855f7',
  legendary: '#f59e0b',
};

const STAGE_LABELS: Record<number, string> = {
  1: 'Новобранец',
  2: 'Искатель',
  3: 'Ветеран',
  4: 'Мастер',
  5: 'Легенда',
};

// ── small presentational helpers ────────────────────────────────────────────

interface ToggleRowProps<T extends string | number> {
  label: string;
  options: readonly T[];
  value: T;
  onChange: (value: T) => void;
  render: (option: T) => string;
}

function ToggleRow<T extends string | number>({
  label,
  options,
  value,
  onChange,
  render,
}: ToggleRowProps<T>) {
  return (
    <div className="stack" style={{ gap: 'var(--space-2)' }}>
      <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
        {label}
      </span>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--space-2)' }}>
        {options.map((opt) => (
          <Button
            key={String(opt)}
            size="sm"
            variant={opt === value ? 'primary' : 'secondary'}
            onClick={() => onChange(opt)}
            aria-pressed={opt === value}
          >
            {render(opt)}
          </Button>
        ))}
      </div>
    </div>
  );
}

interface SlotPickerProps {
  slot: EquipSlot;
  items: ManifestItem[];
  selectedId: string | undefined;
  onSelect: (slot: EquipSlot, id: string | undefined) => void;
}

function SlotPicker({ slot, items, selectedId, onSelect }: SlotPickerProps) {
  return (
    <div className="stack" style={{ gap: 'var(--space-2)' }}>
      <span style={{ fontSize: 'var(--text-sm)', fontWeight: 600 }}>{SLOT_LABELS[slot]}</span>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 'var(--space-2)' }}>
        <Button
          size="sm"
          variant={selectedId === undefined ? 'primary' : 'ghost'}
          onClick={() => onSelect(slot, undefined)}
          aria-pressed={selectedId === undefined}
        >
          —
        </Button>
        {items.map((item) => {
          const active = item.id === selectedId;
          return (
            <Button
              key={item.id}
              size="sm"
              variant={active ? 'primary' : 'secondary'}
              onClick={() => onSelect(slot, active ? undefined : item.id)}
              aria-pressed={active}
              style={{
                borderLeft: `3px solid ${RARITY_COLOR[item.rarity]}`,
              }}
            >
              <span style={{ color: RARITY_COLOR[item.rarity], fontWeight: 700 }}>
                {RARITY_LABELS[item.rarity]}
              </span>
              <span style={{ opacity: 0.85 }}>{item.id.replace(/_/g, ' ')}</span>
            </Button>
          );
        })}
        {items.length === 0 && (
          <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
            нет ассетов
          </span>
        )}
      </div>
    </div>
  );
}

// ── gallery (consistency grid) ───────────────────────────────────────────────

function ConsistencyGallery({ manifest }: { manifest: CharacterManifest }) {
  const stages = manifest.rankStages;
  return (
    <div className="stack" style={{ gap: 'var(--space-3)' }}>
      <span style={{ fontWeight: 700 }}>Галерея консистентности — класс × стадия</span>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={{ padding: 'var(--space-2)', textAlign: 'left' }} />
              {stages.map((stage) => (
                <th
                  key={stage}
                  className="muted"
                  style={{ padding: 'var(--space-2)', fontSize: 'var(--text-sm)', fontWeight: 600 }}
                >
                  {stage}. {STAGE_LABELS[stage] ?? `Стадия ${stage}`}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {manifest.classes.map((cls) => (
              <tr key={cls}>
                <th
                  style={{
                    padding: 'var(--space-2)',
                    textAlign: 'left',
                    fontSize: 'var(--text-sm)',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {CLASS_LABELS[cls]}
                </th>
                {stages.map((stage) => (
                  <td key={stage} style={{ padding: 'var(--space-2)' }}>
                    <PixelFrame style={{ width: 64, height: 96, margin: '0 auto' }}>
                      <CharacterCanvas
                        scale={1}
                        overrides={{ class: cls, stage }}
                        animate={false}
                        ariaLabel={`${CLASS_LABELS[cls]}, стадия ${stage}`}
                        style={{ width: 64, height: 96 }}
                      />
                    </PixelFrame>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ── screen ────────────────────────────────────────────────────────────────────

export default function LabScreen() {
  const { data: manifest, isPending, isError } = useCharacterAssets();

  const [cls, setCls] = useState<CharacterClass>('sage');
  const [stage, setStage] = useState<number>(3);
  const [equipped, setEquipped] = useState<EquippedItemMap>({});
  const [showGallery, setShowGallery] = useState(false);

  const handleSelect = (slot: EquipSlot, id: string | undefined) => {
    setEquipped((prev) => {
      const next = { ...prev };
      if (id === undefined) {
        delete next[slot];
      } else {
        next[slot] = id;
      }
      return next;
    });
  };

  const slotItems = useMemo(() => {
    if (!manifest) return {} as Record<EquipSlot, ManifestItem[]>;
    const map = {} as Record<EquipSlot, ManifestItem[]>;
    for (const slot of RENDERED_SLOTS) {
      map[slot] = itemsForSlot(manifest, slot);
    }
    return map;
  }, [manifest]);

  if (isError) {
    return (
      <div className="stack">
        <h1>Лаборатория</h1>
        <p className="muted">Не удалось загрузить манифест ассетов персонажа.</p>
      </div>
    );
  }

  if (isPending || !manifest) {
    return (
      <div className="stack">
        <h1>Лаборатория</h1>
        <p className="muted">Загрузка манифеста…</p>
      </div>
    );
  }

  const classes = manifest.classes;
  const stages = manifest.rankStages;
  const equippedCount = Object.keys(equipped).length;

  return (
    <div className="stack" style={{ gap: 'var(--space-4)' }}>
      <header className="stack" style={{ gap: 'var(--space-1)' }}>
        <h1>Character Lab</h1>
        <p className="muted" style={{ fontSize: 'var(--text-sm)' }}>
          Инструмент контроля консистентности пиксель-арта (docs/12 §12).
        </p>
      </header>

      {/* Live preview */}
      <Panel className="stack" style={{ gap: 'var(--space-3)', alignItems: 'center' }}>
        <PixelFrame style={{ padding: 'var(--space-3)' }}>
          <CharacterCanvas
            scale={2}
            overrides={{ class: cls, stage, equippedItems: equipped }}
            ariaLabel={`Превью: ${CLASS_LABELS[cls]}, стадия ${stage}`}
          />
        </PixelFrame>
        <div className="row" style={{ gap: 'var(--space-3)', flexWrap: 'wrap', justifyContent: 'center' }}>
          <span style={{ fontWeight: 700 }}>{CLASS_LABELS[cls]}</span>
          <span className="muted">·</span>
          <span>{STAGE_LABELS[stage] ?? `Стадия ${stage}`}</span>
          <span className="muted">·</span>
          <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
            {equippedCount} {equippedCount === 1 ? 'предмет' : 'предметов'}
          </span>
        </div>
      </Panel>

      {/* Controls */}
      <Panel className="stack" style={{ gap: 'var(--space-4)' }}>
        <ToggleRow<CharacterClass>
          label="Класс"
          options={classes}
          value={cls}
          onChange={setCls}
          render={(c) => CLASS_LABELS[c]}
        />
        <ToggleRow<number>
          label="Стадия ранга"
          options={stages}
          value={stage}
          onChange={setStage}
          render={(s) => `${s} · ${STAGE_LABELS[s] ?? s}`}
        />

        <div className="stack" style={{ gap: 'var(--space-3)' }}>
          <div className="row-between">
            <span className="muted" style={{ fontSize: 'var(--text-sm)' }}>
              Экипировка по слотам
            </span>
            <Button size="sm" variant="ghost" onClick={() => setEquipped({})}>
              Очистить всё
            </Button>
          </div>
          {RENDERED_SLOTS.map((slot) => (
            <SlotPicker
              key={slot}
              slot={slot}
              items={slotItems[slot] ?? []}
              selectedId={equipped[slot]}
              onSelect={handleSelect}
            />
          ))}
        </div>
      </Panel>

      {/* Gallery toggle */}
      <Panel className="stack" style={{ gap: 'var(--space-3)' }}>
        <Button
          variant="secondary"
          fullWidth
          onClick={() => setShowGallery((v) => !v)}
          aria-expanded={showGallery}
        >
          {showGallery ? 'Скрыть галерею' : 'Показать галерею консистентности'}
        </Button>
        {showGallery && <ConsistencyGallery manifest={manifest} />}
      </Panel>
    </div>
  );
}
