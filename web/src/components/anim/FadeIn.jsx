import { useEffect, useState } from 'react';

// FadeIn — entrance animation (reactbits.dev style). Fades + slides children in.
export default function FadeIn({ children, delay = 0, y = 16, className = '', as: Tag = 'div' }) {
  const [shown, setShown] = useState(false);
  useEffect(() => {
    const id = requestAnimationFrame(() => setShown(true));
    return () => cancelAnimationFrame(id);
  }, []);
  return (
    <Tag
      className={`anim-fade ${className}`}
      style={{
        transitionDelay: `${delay}ms`,
        opacity: shown ? 1 : 0,
        transform: shown ? 'translateY(0)' : `translateY(${y}px)`,
      }}
    >
      {children}
    </Tag>
  );
}
