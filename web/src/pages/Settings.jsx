import { useTranslation } from 'react-i18next';
import { Card } from '../components/ui';
import LanguageSwitcher from '../components/LanguageSwitcher';
import FadeIn from '../components/anim/FadeIn';

export default function Settings() {
  const { t } = useTranslation();
  return (
    <FadeIn className="stack-lg">
      <Card title={t('settings.language')}>
        <p className="muted">{t('settings.languageDesc')}</p>
        <div className="settings-lang">
          <LanguageSwitcher />
        </div>
      </Card>

      <Card title={t('settings.pricing')}>
        <p className="muted">{t('settings.pricingDesc')}</p>
      </Card>

      <Card title={t('settings.about')}>
        <p>{t('settings.aboutText')}</p>
      </Card>
    </FadeIn>
  );
}
