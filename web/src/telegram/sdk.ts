/**
 * Telegram Mini App SDK wrapper (docs/10).
 *
 * The official telegram-web-app.js is loaded in index.html and exposes
 * window.Telegram.WebApp. This module:
 *  - initialises the WebApp (ready/expand),
 *  - maps themeParams → CSS custom properties so UI matches Telegram theme,
 *  - exposes helpers: hapticFeedback, getInitData, getTimezone.
 *
 * Everything degrades gracefully when running in a plain browser (dev fallback),
 * where window.Telegram is undefined.
 */

// ──────────────────────────────────────────────────────────────────────────
// Minimal typings for the Telegram WebApp object (subset we use).
// ──────────────────────────────────────────────────────────────────────────

export interface TelegramThemeParams {
  bg_color?: string;
  text_color?: string;
  hint_color?: string;
  link_color?: string;
  button_color?: string;
  button_text_color?: string;
  secondary_bg_color?: string;
  header_bg_color?: string;
  accent_text_color?: string;
  section_bg_color?: string;
  section_header_text_color?: string;
  subtitle_text_color?: string;
  destructive_text_color?: string;
}

export type HapticImpactStyle = 'light' | 'medium' | 'heavy' | 'rigid' | 'soft';
export type HapticNotificationType = 'error' | 'success' | 'warning';

export interface TelegramHapticFeedback {
  impactOccurred(style: HapticImpactStyle): void;
  notificationOccurred(type: HapticNotificationType): void;
  selectionChanged(): void;
}

export interface TelegramWebApp {
  initData: string;
  initDataUnsafe: Record<string, unknown>;
  colorScheme: 'light' | 'dark';
  themeParams: TelegramThemeParams;
  isExpanded: boolean;
  viewportHeight: number;
  viewportStableHeight: number;
  HapticFeedback?: TelegramHapticFeedback;
  ready(): void;
  expand(): void;
  setHeaderColor?(color: string): void;
  setBackgroundColor?(color: string): void;
  onEvent(event: string, handler: () => void): void;
  offEvent(event: string, handler: () => void): void;
}

declare global {
  interface Window {
    Telegram?: {
      WebApp?: TelegramWebApp;
    };
  }
}

/** Returns the live WebApp object, or undefined outside Telegram. */
export function getWebApp(): TelegramWebApp | undefined {
  return typeof window !== 'undefined' ? window.Telegram?.WebApp : undefined;
}

/** True when running inside the Telegram client (initData present). */
export function isTelegram(): boolean {
  const wa = getWebApp();
  return Boolean(wa && wa.initData && wa.initData.length > 0);
}

// ──────────────────────────────────────────────────────────────────────────
// Theme → CSS variables
// ──────────────────────────────────────────────────────────────────────────

const THEME_VAR_MAP: Record<keyof TelegramThemeParams, string> = {
  bg_color: '--tg-bg',
  text_color: '--tg-text',
  hint_color: '--tg-hint',
  link_color: '--tg-link',
  button_color: '--tg-button',
  button_text_color: '--tg-button-text',
  secondary_bg_color: '--tg-secondary-bg',
  header_bg_color: '--tg-header-bg',
  accent_text_color: '--tg-accent-text',
  section_bg_color: '--tg-section-bg',
  section_header_text_color: '--tg-section-header-text',
  subtitle_text_color: '--tg-subtitle-text',
  destructive_text_color: '--tg-destructive-text',
};

/** Apply Telegram themeParams to :root as CSS custom properties. */
export function applyThemeParams(params: TelegramThemeParams | undefined): void {
  if (!params || typeof document === 'undefined') return;
  const root = document.documentElement;
  (Object.keys(THEME_VAR_MAP) as Array<keyof TelegramThemeParams>).forEach((key) => {
    const value = params[key];
    if (value) {
      root.style.setProperty(THEME_VAR_MAP[key], value);
    }
  });
}

// ──────────────────────────────────────────────────────────────────────────
// Init
// ──────────────────────────────────────────────────────────────────────────

let initialised = false;

/**
 * Initialise the Telegram WebApp: ready(), expand(), apply theme, and keep
 * CSS vars in sync on theme changes. Idempotent and safe outside Telegram.
 */
export function initTelegram(): void {
  if (initialised) return;
  initialised = true;

  const wa = getWebApp();
  if (!wa) {
    // Dev fallback: nothing to init; tokens.css provides default vars.
    return;
  }

  wa.ready();
  wa.expand();
  applyThemeParams(wa.themeParams);

  const onThemeChanged = (): void => applyThemeParams(getWebApp()?.themeParams);
  wa.onEvent('themeChanged', onThemeChanged);
}

// ──────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────

/**
 * Raw initData string for the Authorization header. Empty string outside
 * Telegram (caller falls back to X-Device-Id).
 */
export function getInitData(): string {
  return getWebApp()?.initData ?? '';
}

/** IANA timezone resolved from the browser (docs/10 §4). */
export function getTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
  } catch {
    return 'UTC';
  }
}

export type HapticType = HapticImpactStyle | HapticNotificationType | 'selection';

const NOTIFICATION_TYPES: ReadonlySet<string> = new Set<HapticNotificationType>([
  'error',
  'success',
  'warning',
]);

/**
 * Fire Telegram haptic feedback. Accepts impact styles (light/medium/heavy/
 * rigid/soft), notification types (error/success/warning) or 'selection'.
 * No-op outside Telegram.
 */
export function hapticFeedback(type: HapticType = 'medium'): void {
  const haptic = getWebApp()?.HapticFeedback;
  if (!haptic) return;

  if (type === 'selection') {
    haptic.selectionChanged();
    return;
  }
  if (NOTIFICATION_TYPES.has(type)) {
    haptic.notificationOccurred(type as HapticNotificationType);
    return;
  }
  haptic.impactOccurred(type as HapticImpactStyle);
}
