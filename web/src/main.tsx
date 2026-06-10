/**
 * App entry: mounts React with QueryClientProvider + BrowserRouter, and
 * initialises the Telegram WebApp (ready/expand + theme → CSS vars).
 *
 * Robustness: a Telegram Mini App WebView has no visible console, so an
 * uncaught error during boot is a silent white screen. We therefore (a) guard
 * initTelegram, (b) install global error/rejection handlers that paint the
 * error onto #root, and (c) wrap <App/> in an ErrorBoundary. See docs/10.
 */

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import { ErrorBoundary } from './ErrorBoundary';
import { queryClient } from './lib/query';
import { initTelegram } from './telegram/sdk';
import './styles/globals.css';

/** Paint a fatal error straight into the DOM (visible inside Telegram). */
function showFatal(title: string, detail: string): void {
  const root = document.getElementById('root') ?? document.body;
  const pre = document.createElement('pre');
  pre.style.cssText =
    'white-space:pre-wrap;word-break:break-word;padding:16px;margin:0;min-height:100vh;background:#111316;color:#ff8a8a;font:12px/1.5 ui-monospace,Menlo,monospace';
  pre.textContent = `💥 ${title}\n\n${detail}`;
  root.replaceChildren(pre);
}

// Catch errors thrown OUTSIDE React (e.g. during initTelegram, or async).
// Ignore resource-load errors (a 404'd <img>/<script> also fires 'error' but
// is not an ErrorEvent with a real .error) so a broken sprite never wipes the UI.
window.addEventListener('error', (e) => {
  if (!(e instanceof ErrorEvent) || !e.error) return;
  showFatal('window error', e.error.stack ?? String(e.error));
});
window.addEventListener('unhandledrejection', (e) => {
  const r = e.reason;
  showFatal('unhandled promise rejection', r?.stack ?? String(r));
});

try {
  initTelegram();
} catch (e) {
  showFatal('initTelegram() threw', (e as Error)?.stack ?? String(e));
}

const container = document.getElementById('root');
if (!container) {
  throw new Error('Root container #root not found');
}

createRoot(container).render(
  <StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <App />
        </BrowserRouter>
      </QueryClientProvider>
    </ErrorBoundary>
  </StrictMode>,
);
