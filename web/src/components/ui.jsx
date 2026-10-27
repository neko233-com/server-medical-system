import { useEffect } from 'react';
import Reicon from './Reicon';

export function Button({ variant = 'primary', size = 'md', icon, children, className = '', ...props }) {
  return (
    <button className={`btn btn--${variant} btn--${size} ${className}`} {...props}>
      {icon && <Reicon name={icon} size={size === 'sm' ? 15 : 17} />}
      {children && <span>{children}</span>}
    </button>
  );
}

export function Card({ title, actions, children, className = '', padding = true }) {
  return (
    <div className={`card ${className}`}>
      {(title || actions) && (
        <div className="card__head">
          {title && <h3 className="card__title">{title}</h3>}
          {actions && <div className="card__actions">{actions}</div>}
        </div>
      )}
      <div className={padding ? 'card__body' : 'card__body card__body--flush'}>{children}</div>
    </div>
  );
}

export function Badge({ tone = 'neutral', children }) {
  return <span className={`badge badge--${tone}`}>{children}</span>;
}

export function Spinner({ size = 18 }) {
  return (
    <span className="spinner" style={{ width: size, height: size }} aria-label="loading" />
  );
}

export function Empty({ icon = 'Box', text }) {
  return (
    <div className="empty">
      <Reicon name={icon} size={34} color="var(--muted)" />
      <p>{text}</p>
    </div>
  );
}

export function Field({ label, children, optional, className = '' }) {
  return (
    <label className={`field ${className}`}>
      {label && (
        <span className="field__label">
          {label}
          {optional && <em className="field__opt">（{optional}）</em>}
        </span>
      )}
      {children}
    </label>
  );
}

export function Input({ className = '', ...props }) {
  return <input className={`input ${className}`} {...props} />;
}

export function Textarea({ className = '', ...props }) {
  return <textarea className={`input textarea ${className}`} {...props} />;
}

export function Select({ className = '', children, ...props }) {
  return (
    <select className={`input select ${className}`} {...props}>
      {children}
    </select>
  );
}

export function Toggle({ checked, onChange, label }) {
  return (
    <label className="toggle">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span className="toggle__track" />
      {label && <span className="toggle__label">{label}</span>}
    </label>
  );
}

// Generic table. columns: [{key, label, render?, align?}]
export function Table({ columns, rows, rowKey = 'id', empty }) {
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c.key} className={c.align ? `ta-${c.align}` : ''}>
                {c.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row[rowKey] ?? JSON.stringify(row)}>
              {columns.map((c) => (
                <td key={c.key} className={c.align ? `ta-${c.align}` : ''}>
                  {c.render ? c.render(row) : row[c.key]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 && empty && <Empty text={empty} />}
    </div>
  );
}

export function Modal({ open, onClose, title, children, footer, width = 520 }) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e) => e.key === 'Escape' && onClose?.();
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="modal-overlay" onMouseDown={onClose}>
      <div className="modal" style={{ maxWidth: width }} onMouseDown={(e) => e.stopPropagation()}>
        <div className="modal__head">
          <h3>{title}</h3>
          <button className="modal__close" onClick={onClose} aria-label="close">
            <Reicon name="X" size={18} />
          </button>
        </div>
        <div className="modal__body">{children}</div>
        {footer && <div className="modal__foot">{footer}</div>}
      </div>
    </div>
  );
}

export function StatCard({ label, value, icon, tone = 'primary', hint }) {
  return (
    <div className={`stat stat--${tone}`}>
      <div className="stat__icon">
        <Reicon name={icon} size={22} />
      </div>
      <div className="stat__meta">
        <span className="stat__value">{value}</span>
        <span className="stat__label">{label}</span>
        {hint && <span className="stat__hint">{hint}</span>}
      </div>
    </div>
  );
}
