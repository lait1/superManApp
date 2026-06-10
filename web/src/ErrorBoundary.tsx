/**
 * ErrorBoundary — renders the actual error onto the page instead of a silent
 * white screen. Critical for Telegram Mini Apps, where the WebView has no
 * visible console: without this, any render-time throw is an unobservable
 * blank screen (see docs/10). Keep this mounted around the whole app.
 */

import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

const boxStyle: React.CSSProperties = {
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
  padding: '16px',
  margin: 0,
  minHeight: '100vh',
  background: '#111316',
  color: '#ff8a8a',
  font: '12px/1.5 ui-monospace, SFMono-Regular, Menlo, monospace',
};

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Also log for the (rare) case a console is attached.
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  render(): ReactNode {
    const { error } = this.state;
    if (error) {
      return (
        <pre style={boxStyle}>
          {`💥 Приложение упало при рендере:\n\n${error.name}: ${error.message}\n\n${error.stack ?? ''}`}
        </pre>
      );
    }
    return this.props.children;
  }
}
