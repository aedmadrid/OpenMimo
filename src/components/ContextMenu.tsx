import React, { useEffect, useRef } from 'react';

export interface ContextMenuItem {
  label: string;
  shortcut?: string;
  danger?: boolean;
  disabled?: boolean;
  divider?: boolean;
  onClick?: () => void;
}

interface Props {
  x: number;
  y: number;
  items: ContextMenuItem[];
  onClose: () => void;
}

export const ContextMenu: React.FC<Props> = ({ x, y, items, onClose }) => {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    };
    window.addEventListener('keydown', onKey);
    window.addEventListener('mousedown', onDown);
    return () => {
      window.removeEventListener('keydown', onKey);
      window.removeEventListener('mousedown', onDown);
    };
  }, [onClose]);

  // Keep menu comfortably inside viewport bounds
  const clampedX = Math.min(x, window.innerWidth - 235);
  const clampedY = Math.min(y, window.innerHeight - Math.min(items.length * 32 + 16, 360));

  return (
    <div
      ref={ref}
      className="ctx-menu"
      style={{ left: Math.max(8, clampedX), top: Math.max(8, clampedY) }}
      onContextMenu={(e) => e.preventDefault()}
    >
      {items.map((item, idx) =>
        item.divider ? (
          <div key={idx} className="menu-sep" />
        ) : (
          <button
            key={idx}
            className={`menu-item ${item.danger ? 'ctx-danger' : ''}`}
            disabled={item.disabled}
            onClick={() => {
              onClose();
              item.onClick?.();
            }}
          >
            <span>{item.label}</span>
            {item.shortcut && <span style={{ opacity: 0.55, fontSize: 11 }}>{item.shortcut}</span>}
          </button>
        )
      )}
    </div>
  );
};
