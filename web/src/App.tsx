/**
 * App — router shell (docs/05 §1).
 *
 * Declares ALL routes via lazy imports on FIXED paths. Other agents implement
 * the screen components at these paths (overwriting the Phase-1 stubs); they
 * MUST NOT edit this file.
 *
 *   '/'        → screens/Hero/HeroScreen
 *   '/quests'  → screens/Quests/QuestsScreen
 *   '/shop'    → screens/Shop/ShopScreen
 *   '/profile' → screens/Profile/ProfileScreen
 *   '/lab'     → screens/Lab/LabScreen
 *
 * Layout: routed screen in <main>, persistent bottom TabBar with the check-in
 * FAB, and the CheckinModal rendered above everything (state-driven via store).
 */

import { Suspense, lazy } from 'react';
import { Route, Routes } from 'react-router-dom';
import { TabBar } from './components/ui';
import { useStore } from './store/useStore';
import { useMe } from './lib/query';
import CheckinModal from './screens/Checkin/CheckinModal';

const HeroScreen = lazy(() => import('./screens/Hero/HeroScreen'));
const QuestsScreen = lazy(() => import('./screens/Quests/QuestsScreen'));
const ShopScreen = lazy(() => import('./screens/Shop/ShopScreen'));
const ProfileScreen = lazy(() => import('./screens/Profile/ProfileScreen'));
const LabScreen = lazy(() => import('./screens/Lab/LabScreen'));
const OnboardingScreen = lazy(() => import('./screens/Onboarding/OnboardingScreen'));

function ScreenFallback() {
  return (
    <div className="muted" style={{ padding: 'var(--space-6)', textAlign: 'center' }}>
      Загрузка…
    </div>
  );
}

export default function App() {
  const openCheckin = useStore((s) => s.openCheckin);
  const { data: me } = useMe();

  // First-run gate: until the hero is named & styled, the whole shell is the
  // onboarding flow (no tabs, no routes — RPG "create your character" moment).
  if (me && !me.character.onboarded) {
    return (
      <div className="app-shell">
        <main className="app-content">
          <Suspense fallback={<ScreenFallback />}>
            <OnboardingScreen />
          </Suspense>
        </main>
      </div>
    );
  }

  return (
    <div className="app-shell">
      <main className="app-content">
        <Suspense fallback={<ScreenFallback />}>
          <Routes>
            <Route path="/" element={<HeroScreen />} />
            <Route path="/quests" element={<QuestsScreen />} />
            <Route path="/shop" element={<ShopScreen />} />
            <Route path="/profile" element={<ProfileScreen />} />
            <Route path="/lab" element={<LabScreen />} />
          </Routes>
        </Suspense>
      </main>

      <TabBar onCheckin={openCheckin} />

      {/* Check-in modal lives above routes, controlled by store state. */}
      <CheckinModal />
    </div>
  );
}
