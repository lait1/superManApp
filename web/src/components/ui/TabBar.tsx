/**
 * TabBar — bottom navigation (docs/05 §1) with a floating central check-in FAB.
 *
 * Tabs (4): Hero / Quests / Shop / Profile. The Lab route ('/lab') has no tab
 * and is reachable programmatically. The FAB opens the check-in modal via the
 * store (onCheckin handler passed from App).
 *
 * Uses react-router NavLink for active styling. Tab definitions live here as
 * the single source of nav metadata, keyed to the fixed route paths in App.tsx.
 */

import type { CSSProperties } from 'react';
import { NavLink } from 'react-router-dom';

export interface TabBarProps {
  /** Opens the check-in modal (FAB). */
  onCheckin: () => void;
}

interface TabDef {
  to: string;
  icon: string;
  label: string;
}

export const TABS: readonly TabDef[] = [
  { to: '/', icon: '🏠', label: 'Герой' },
  { to: '/quests', icon: '⚔️', label: 'Квесты' },
  { to: '/shop', icon: '🏪', label: 'Магазин' },
  { to: '/profile', icon: '👤', label: 'Профиль' },
];

const barStyle: CSSProperties = {
  position: 'fixed',
  left: 0,
  right: 0,
  bottom: 0,
  zIndex: 'var(--z-tabbar)' as unknown as number,
  height: 'calc(var(--tabbar-height) + var(--safe-bottom))',
  paddingBottom: 'var(--safe-bottom)',
  background: 'var(--tg-header-bg)',
  borderTop: '1px solid var(--tg-secondary-bg)',
  display: 'grid',
  gridTemplateColumns: 'repeat(4, 1fr)',
};

const fabStyle: CSSProperties = {
  position: 'fixed',
  left: '50%',
  transform: 'translateX(-50%)',
  bottom: 'calc(var(--tabbar-height) + var(--safe-bottom) - var(--fab-size) / 2)',
  zIndex: 'var(--z-fab)' as unknown as number,
  width: 'var(--fab-size)',
  height: 'var(--fab-size)',
  borderRadius: 'var(--radius-pill)',
  background: 'var(--tg-button)',
  color: 'var(--tg-button-text)',
  fontSize: 'var(--text-2xl)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  boxShadow: 'var(--shadow-fab)',
};

function tabItemStyle(isActive: boolean): CSSProperties {
  return {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '2px',
    fontSize: 'var(--text-xs)',
    color: isActive ? 'var(--tg-accent-text)' : 'var(--tg-hint)',
    height: 'var(--tabbar-height)',
  };
}

export function TabBar({ onCheckin }: TabBarProps) {
  return (
    <>
      <button aria-label="Отметить" onClick={onCheckin} style={fabStyle}>
        ⊕
      </button>
      <nav style={barStyle}>
        {TABS.map((tab) => (
          <NavLink
            key={tab.to}
            to={tab.to}
            end={tab.to === '/'}
            style={({ isActive }) => tabItemStyle(isActive)}
          >
            <span aria-hidden style={{ fontSize: 'var(--text-lg)' }}>
              {tab.icon}
            </span>
            <span>{tab.label}</span>
          </NavLink>
        ))}
      </nav>
    </>
  );
}
