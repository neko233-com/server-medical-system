// Central registry of supported languages. Extend this array to expose a new
// locale in the language switcher — the matching resource must exist in i18n/index.js.
export const LANGUAGES = [
  { code: 'zh-CN', label: '中文', short: '中' },
  { code: 'en-US', label: 'English', short: 'EN' },
];

export const DEFAULT_LANG = 'zh-CN';
