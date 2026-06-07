/**
 * useReducedMotion — true when the user prefers reduced motion
 * (`prefers-reduced-motion: reduce`). The reward layer (docs/05 §6 / docs/12 §12)
 * must honour this: confetti / particles / springy bounces are suppressed and
 * effects degrade to instant fades.
 */

import { useEffect, useState } from 'react';

const QUERY = '(prefers-reduced-motion: reduce)';

export function useReducedMotion(): boolean {
  const [reduced, setReduced] = useState<boolean>(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return false;
    return window.matchMedia(QUERY).matches;
  });

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const mql = window.matchMedia(QUERY);
    const onChange = (e: MediaQueryListEvent): void => setReduced(e.matches);
    // Safari < 14 lacks addEventListener on MediaQueryList.
    if (typeof mql.addEventListener === 'function') {
      mql.addEventListener('change', onChange);
      return () => mql.removeEventListener('change', onChange);
    }
    mql.addListener(onChange);
    return () => mql.removeListener(onChange);
  }, []);

  return reduced;
}
