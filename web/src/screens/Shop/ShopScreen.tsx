/**
 * ShopScreen — '/shop' (docs/04 §7, docs/05 §5).
 *
 * Sections: Снаряжение / Косметика / Расходники (grouped from useShop by slot),
 * plus an Инвентарь tab (useInventory) for «Надеть» / «Снять».
 *
 *  - Each shop row shows the rarity-coloured name, price + 💰 and a «Купить»
 *    button (POST /shop/{id}/buy). A 409 `insufficient_gold` surfaces a «мало
 *    золота» message instead of a generic error.
 *  - Inventory rows show equipped state with «Надеть» / «Снять» actions.
 *
 * Loading / error / empty states are handled per tab.
 */

import { useMemo, useState } from 'react';
import type { CSSProperties } from 'react';
import { Button, Panel } from '../../components/ui';
import {
  useBuyItem,
  useEquipItem,
  useInventory,
  useShop,
  useUnequipItem,
} from '../../lib/query';
import { useStore } from '../../store/useStore';
import { hapticFeedback } from '../../telegram/sdk';
import type { InventoryItem, ShopItem } from '../../types/api';
import {
  formatGold,
  itemIcon,
  RARITY_COLOR,
  RARITY_LABEL,
  SLOT_LABEL,
  slotSection,
  type ShopSection,
} from '../Settings/itemMeta';
import { EmptyState, ErrorState, LoadingState, SectionHeader } from '../Settings/states';
import { Tabs, type TabOption } from '../Settings/Tabs';

type ShopTab = ShopSection | 'inventory';

const TAB_OPTIONS: ReadonlyArray<TabOption<ShopTab>> = [
  { key: 'gear', label: 'Снаряжение' },
  { key: 'cosmetic', label: 'Косметика' },
  { key: 'consumable', label: 'Расходники' },
  { key: 'inventory', label: 'Инвентарь' },
];

const SECTION_TITLE: Record<ShopSection, string> = {
  gear: 'Снаряжение',
  cosmetic: 'Косметика',
  consumable: 'Расходники',
};

const rowStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 'var(--space-3)',
};

function GoldBadge({ gold }: { gold: number | null }) {
  if (gold === null) return null;
  return (
    <span className="muted" style={{ fontSize: 'var(--text-sm)', whiteSpace: 'nowrap' }}>
      💰 {formatGold(gold)}
    </span>
  );
}

function ItemMeta({ name, color, sub }: { name: string; color: string; sub: string }) {
  return (
    <div style={{ minWidth: 0, flex: 1 }}>
      <div
        style={{
          fontWeight: 'var(--weight-medium)' as unknown as number,
          color,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
        }}
      >
        {name}
      </div>
      <div className="muted" style={{ fontSize: 'var(--text-xs)' }}>
        {sub}
      </div>
    </div>
  );
}

function ShopRow({ item }: { item: ShopItem }) {
  const buy = useBuyItem();
  const color = RARITY_COLOR[item.rarity];

  const canBuy = item.purchasable && item.price !== null;
  const lowGold = buy.isError && buy.error.code === 'insufficient_gold';

  function onBuy(): void {
    buy.mutate(item.id, {
      onSuccess: () => hapticFeedback('success'),
      onError: () => hapticFeedback('error'),
    });
  }

  return (
    <Panel className="stack" style={{ gap: 'var(--space-2)' }}>
      <div style={rowStyle}>
        <span aria-hidden style={{ fontSize: 'var(--text-xl)' }}>
          {itemIcon(item.slot, item.icon)}
        </span>
        <ItemMeta
          name={item.name}
          color={color}
          sub={`${RARITY_LABEL[item.rarity]} · ${SLOT_LABEL[item.slot]}`}
        />
        <GoldBadge gold={item.price} />
        {canBuy ? (
          <Button size="sm" onClick={onBuy} disabled={buy.isPending}>
            {buy.isPending ? '…' : 'Купить'}
          </Button>
        ) : (
          <span className="muted" style={{ fontSize: 'var(--text-xs)', whiteSpace: 'nowrap' }}>
            за квест
          </span>
        )}
      </div>
      {buy.isError && (
        <span
          role="alert"
          style={{ color: 'var(--tg-destructive-text)', fontSize: 'var(--text-sm)' }}
        >
          {lowGold ? 'Недостаточно золота' : buy.error.message}
        </span>
      )}
    </Panel>
  );
}

function InventoryRow({ item }: { item: InventoryItem }) {
  const equip = useEquipItem();
  const unequip = useUnequipItem();
  const busy = equip.isPending || unequip.isPending;
  const color = RARITY_COLOR[item.rarity];
  const isConsumable = item.slot === 'consumable';

  function onEquip(): void {
    equip.mutate(item.id, {
      onSuccess: () => hapticFeedback('success'),
      onError: () => hapticFeedback('error'),
    });
  }
  function onUnequip(): void {
    unequip.mutate(item.id, {
      onSuccess: () => hapticFeedback('light'),
      onError: () => hapticFeedback('error'),
    });
  }

  const sub = `${RARITY_LABEL[item.rarity]} · ${SLOT_LABEL[item.slot]}${
    item.quantity > 1 ? ` ×${item.quantity}` : ''
  }`;

  const error = equip.error ?? unequip.error;

  return (
    <Panel className="stack" style={{ gap: 'var(--space-2)' }}>
      <div style={rowStyle}>
        <span aria-hidden style={{ fontSize: 'var(--text-xl)' }}>
          {itemIcon(item.slot, item.icon)}
        </span>
        <ItemMeta name={item.name} color={color} sub={sub} />
        {item.equipped && (
          <span style={{ color: 'var(--rarity-uncommon)', fontSize: 'var(--text-xs)' }}>
            надето
          </span>
        )}
        {isConsumable ? (
          <span className="muted" style={{ fontSize: 'var(--text-xs)', whiteSpace: 'nowrap' }}>
            расходник
          </span>
        ) : item.equipped ? (
          <Button size="sm" variant="secondary" onClick={onUnequip} disabled={busy}>
            Снять
          </Button>
        ) : (
          <Button size="sm" onClick={onEquip} disabled={busy}>
            Надеть
          </Button>
        )}
      </div>
      {(equip.isError || unequip.isError) && error && (
        <span
          role="alert"
          style={{ color: 'var(--tg-destructive-text)', fontSize: 'var(--text-sm)' }}
        >
          {error.message}
        </span>
      )}
    </Panel>
  );
}

function ShopList({ section }: { section: ShopSection }) {
  const { data, isPending, isError, error, refetch } = useShop();

  const items = useMemo<ShopItem[]>(() => {
    if (!data) return [];
    return data.filter((it) => slotSection(it.slot) === section);
  }, [data, section]);

  if (isPending) return <LoadingState />;
  if (isError) return <ErrorState error={error} onRetry={() => void refetch()} />;
  if (items.length === 0) {
    return <EmptyState icon="🛒" title="Пусто" hint={`В разделе «${SECTION_TITLE[section]}» пока нет товаров.`} />;
  }
  return (
    <div className="stack" style={{ gap: 'var(--space-3)' }}>
      <SectionHeader>{SECTION_TITLE[section]}</SectionHeader>
      {items.map((it) => (
        <ShopRow key={it.id} item={it} />
      ))}
    </div>
  );
}

function InventoryList() {
  const { data, isPending, isError, error, refetch } = useInventory();

  if (isPending) return <LoadingState />;
  if (isError) return <ErrorState error={error} onRetry={() => void refetch()} />;
  if (!data || data.length === 0) {
    return <EmptyState icon="🎒" title="Инвентарь пуст" hint="Купи что-нибудь в магазине." />;
  }
  return (
    <div className="stack" style={{ gap: 'var(--space-3)' }}>
      {data.map((it) => (
        <InventoryRow key={it.id} item={it} />
      ))}
    </div>
  );
}

export default function ShopScreen() {
  const [tab, setTab] = useState<ShopTab>('gear');
  const gold = useStore((s) => s.character?.gold ?? null);

  return (
    <div className="stack" style={{ gap: 'var(--space-4)' }}>
      <div className="row-between">
        <h1 style={{ fontSize: 'var(--text-xl)' }}>🏪 Магазин</h1>
        {gold !== null && (
          <span style={{ fontWeight: 'var(--weight-bold)' as unknown as number }}>
            💰 {formatGold(gold)}
          </span>
        )}
      </div>

      <Tabs options={TAB_OPTIONS} value={tab} onChange={setTab} />

      {tab === 'inventory' ? <InventoryList /> : <ShopList section={tab} />}
    </div>
  );
}
