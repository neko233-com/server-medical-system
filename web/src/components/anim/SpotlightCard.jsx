import { useRef } from 'react';

// SpotlightCard — follows the cursor with a soft radial glow (reactbits.dev pattern).
export default function SpotlightCard({ children, className = '' }) {
  const ref = useRef(null);
  const onMove = (e) => {
    const el = ref.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    el.style.setProperty('--mx', `${e.clientX - r.left}px`);
    el.style.setProperty('--my', `${e.clientY - r.top}px`);
  };
  return (
    <div ref={ref} className={`spotlight-card ${className}`} onMouseMove={onMove}>
      <div className="spotlight-card__inner">{children}</div>
    </div>
  );
}
