import { useEffect, useState } from 'react';

// BlurText — staggered blur-to-sharp text reveal (reactbits.dev "Blur Text" pattern).
export default function BlurText({ text, className = '', delayStep = 40 }) {
  const words = String(text).split(' ');
  const [shown, setShown] = useState(false);
  useEffect(() => {
    const id = requestAnimationFrame(() => setShown(true));
    return () => cancelAnimationFrame(id);
  }, []);
  return (
    <span className={className} style={{ display: 'inline-block' }}>
      {words.map((w, i) => (
        <span
          key={i}
          style={{
            display: 'inline-block',
            filter: shown ? 'blur(0)' : 'blur(8px)',
            opacity: shown ? 1 : 0,
            transform: shown ? 'translateY(0)' : 'translateY(8px)',
            transition: `filter .6s ease ${i * delayStep}ms, opacity .6s ease ${i * delayStep}ms, transform .6s ease ${i * delayStep}ms`,
            marginRight: '0.28em',
          }}
        >
          {w}
        </span>
      ))}
    </span>
  );
}
