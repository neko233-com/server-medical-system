import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Reicon from '../Reicon';

const items = [
  { to: '/dashboard', icon: 'Home', key: 'nav.dashboard' },
  { to: '/patients', icon: 'Users', key: 'nav.patients' },
  { to: '/consultation', icon: 'Stethoscope', key: 'nav.consultation' },
  { to: '/medicines', icon: 'Pill', key: 'nav.medicines' },
  { to: '/settings', icon: 'Settings', key: 'nav.settings' },
];

export default function Sidebar() {
  const { t } = useTranslation();
  return (
    <aside className="sidebar">
      <div className="sidebar__brand">
        <span className="sidebar__logo">
          <Reicon name="Heart" size={22} color="#fff" />
        </span>
        <div className="sidebar__brand-text">
          <strong>{t('app.name')}</strong>
          <small>{t('app.tagline')}</small>
        </div>
      </div>
      <nav className="sidebar__nav">
        {items.map((it) => (
          <NavLink
            key={it.to}
            to={it.to}
            className={({ isActive }) => `nav-item ${isActive ? 'is-active' : ''}`}
          >
            <Reicon name={it.icon} size={20} />
            <span>{t(it.key)}</span>
          </NavLink>
        ))}
      </nav>
      <div className="sidebar__foot">
        <small>{t('footer.text')}</small>
      </div>
    </aside>
  );
}
