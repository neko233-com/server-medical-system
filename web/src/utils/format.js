// Helpers for localization-aware display.

// Pick the medicine name matching the active locale. Medicine records carry both
// name_zh and name_en, so no translation tables are needed for drug names.
export function medicineName(med, lang) {
  if (!med) return '';
  if (lang === 'zh-CN' && med.name_zh) return med.name_zh;
  if (lang === 'en-US' && med.name_en) return med.name_en;
  return med.name_zh || med.name_en || '';
}

export function categoryName(med, lang) {
  if (!med) return '';
  if (lang === 'zh-CN' && med.category) return med.category;
  if (lang === 'en-US' && med.category_en) return med.category_en;
  return med.category || med.category_en || '';
}

export function formatMoney(amount, currency = 'CNY') {
  const symbol = currency === 'CNY' ? '¥' : currency === 'USD' ? '$' : '';
  const value = Number(amount || 0).toFixed(2);
  return symbol ? `${symbol}${value}` : `${value} ${currency}`;
}

export function genderLabel(g, t) {
  if (g === 'M') return t('gender.male');
  if (g === 'F') return t('gender.female');
  return g || '-';
}
