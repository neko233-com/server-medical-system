import { Outlet, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Sidebar from './Sidebar';
import Topbar from './Topbar';
import AnimatedBackground from '../anim/AnimatedBackground';

const TITLES = {
  '/dashboard': 'dashboard.title',
  '/patients': 'patients.title',
  '/consultation': 'consultation.title',
  '/medicines': 'medicines.title',
  '/settings': 'settings.title',
};

export default function AppLayout() {
  const { t } = useTranslation();
  const loc = useLocation();
  const titleKey = TITLES[loc.pathname] || 'app.name';

  return (
    <div className="app-shell">
      <AnimatedBackground />
      <Sidebar />
      <div className="app-main">
        <Topbar title={t(titleKey)} />
        <main className="app-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
