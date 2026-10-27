import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import zhCN from './locales/zh-CN';
import enUS from './locales/en-US';

// The active language is persisted to localStorage under `app_lang`.
const saved = localStorage.getItem('app_lang') || 'zh-CN';

// ---------------------------------------------------------------------------
// To add a new language:
//   1. Create src/i18n/locales/<code>.js exporting a flat key->string map.
//   2. Import it above and register it in `resources` below.
//   3. Add the language to LANGUAGES in src/i18n/languages.js (drives the UI).
// No backend changes are required — medicine names already ship bilingual data.
// ---------------------------------------------------------------------------
i18n.use(initReactI18next).init({
  resources: {
    'zh-CN': { translation: zhCN },
    'en-US': { translation: enUS },
  },
  lng: saved,
  fallbackLng: 'zh-CN',
  interpolation: { escapeValue: false },
});

export default i18n;
