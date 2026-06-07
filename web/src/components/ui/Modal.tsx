/**
 * Modal — bottom-sheet style overlay (docs/05 §3: check-in modal).
 * Rendered above routes; animated in/out with Framer Motion. Backdrop click
 * and the close button both call onClose. Respects prefers-reduced-motion via
 * the global CSS override.
 */

import { AnimatePresence, motion } from 'framer-motion';
import type { CSSProperties, ReactNode } from 'react';

export interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: ReactNode;
  children?: ReactNode;
  /** Hide the default close (✕) button. */
  hideClose?: boolean;
}

const backdropStyle: CSSProperties = {
  position: 'fixed',
  inset: 0,
  background: 'rgba(0, 0, 0, 0.55)',
  zIndex: 'var(--z-modal)' as unknown as number,
  display: 'flex',
  alignItems: 'flex-end',
  justifyContent: 'center',
};

const sheetStyle: CSSProperties = {
  width: '100%',
  maxWidth: 'var(--content-max-width)',
  background: 'var(--tg-bg)',
  borderTopLeftRadius: 'var(--radius-lg)',
  borderTopRightRadius: 'var(--radius-lg)',
  padding: 'var(--space-5)',
  paddingBottom: 'calc(var(--space-5) + var(--safe-bottom))',
  boxShadow: 'var(--shadow-2)',
};

export function Modal({ open, onClose, title, children, hideClose = false }: ModalProps) {
  return (
    <AnimatePresence>
      {open && (
        <motion.div
          style={backdropStyle}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.18 }}
          onClick={onClose}
        >
          <motion.div
            style={sheetStyle}
            role="dialog"
            aria-modal="true"
            initial={{ y: '100%' }}
            animate={{ y: 0 }}
            exit={{ y: '100%' }}
            transition={{ type: 'spring', stiffness: 380, damping: 36 }}
            onClick={(e) => e.stopPropagation()}
          >
            {(title || !hideClose) && (
              <div
                className="row-between"
                style={{ marginBottom: 'var(--space-4)' }}
              >
                <div
                  style={{
                    fontSize: 'var(--text-lg)',
                    fontWeight: 'var(--weight-bold)' as unknown as number,
                  }}
                >
                  {title}
                </div>
                {!hideClose && (
                  <button
                    aria-label="Закрыть"
                    onClick={onClose}
                    style={{
                      fontSize: 'var(--text-xl)',
                      color: 'var(--tg-hint)',
                      lineHeight: 1,
                    }}
                  >
                    ✕
                  </button>
                )}
              </div>
            )}
            {children}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
