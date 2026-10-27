import { NavLink } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Reicon from '../Reicon';
import { useAuth } from '../../context/AuthContext';

// Each item declares the permission required to see it (undefined = everyone).
const items = [
  { to: '/dashboard', icon: 'Home', key: 'nav.dashboard' },
  { to: '/patients', icon: 'Users', key: 'nav.patients', perm: 'patient:read' },
  { to: '/consultation', icon: 'Stethoscope', key: 'nav.consultation', perm: 'case:write' },
  { to: '/appointments', icon: 'CalendarCheck', key: 'nav.appointments', perm: 'appointment:read' },
  { to: '/medicines', icon: 'Pill', key: 'nav.medicines', perm: 'medicine:read' },
  { to: '/beds', icon: 'Armchair', key: 'nav.beds', perm: 'bed:read' },
  { to: '/billing', icon: 'Receipt', key: 'nav.billing', perm: 'billing:read' },
  { to: '/rooms', icon: 'Building2', key: 'nav.rooms', perm: 'room:read' },
  { to: '/printers', icon: 'Printer', key: 'nav.printers', perm: 'printer:read' },
  { to: '/surgeries', icon: 'Scissors', key: 'nav.surgeries', perm: 'surgery:read' },
  { to: '/imaging', icon: 'Scan', key: 'nav.imaging', perm: 'imaging:read' },
  { to: '/rbac', icon: 'Shield', key: 'nav.rbac', perm: 'user:read' },
  { to: '/operations', icon: 'Activity', key: 'nav.operations', perm: 'report:read' },
  { to: '/settings', icon: 'Settings', key: 'nav.settings' },
];

export default function Sidebar() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const visible = items.filter((it) => !it.perm || can(it.perm));

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
        {visible.map((it) => (
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
