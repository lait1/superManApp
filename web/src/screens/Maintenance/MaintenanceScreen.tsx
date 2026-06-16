/**
 * MaintenanceScreen — full-screen "under maintenance" gate.
 *
 * Rendered by App INSTEAD of the routed shell when /me responds with the
 * "maintenance" error code (HTTP 503): the backend has MAINTENANCE_MODE on and
 * the current user is not the admin. No tabs, no routes — same shape as the
 * onboarding gate.
 */

import { Panel } from '../../components/ui';

export default function MaintenanceScreen() {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 'var(--space-4)',
        padding: 'var(--space-4)',
        maxWidth: 480,
        margin: '0 auto',
        minHeight: '60vh',
        justifyContent: 'center',
      }}
    >
      <div style={{ textAlign: 'center', fontSize: 56 }}>🛠️</div>
      <h1 style={{ textAlign: 'center', margin: 0 }}>Идут технические работы</h1>
      <Panel>
        <p className="muted" style={{ textAlign: 'center', margin: 0 }}>
          superMen на короткой паузе — мы дорабатываем приложение.
          Загляни чуть позже, твой прогресс на месте.
        </p>
      </Panel>
    </div>
  );
}
