import { useTranslation } from 'react-i18next';
import { LANGUAGES } from '../i18n/languages';
import Reicon from './Reicon';

// Language switcher. Changing it persists to localStorage and updates <html lang>.
export default function LanguageSwitcher() {
  const { i18n } = useTranslation();
  const current = i18n.language;

  const change = (code) => {
    i18n.changeLanguage(code);
    localStorage.setItem('app_lang', code);
    document.documentElement.lang = code;
  };

  return (
    <div className="lang-switch" title="Language">
      <Reicon name="Language" size={16} />
      {LANGUAGES.map((l) => (
        <button
          key={l.code}
          className={`lang-switch__item ${current === l.code ? 'is-active' : ''}`}
          onClick={() => change(l.code)}
        >
          {l.short}
        </button>
      ))}
    </div>
  );
}
