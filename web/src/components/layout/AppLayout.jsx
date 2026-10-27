import { Outlet, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Sidebar from './Sidebar';
import Topbar from './Topbar';
import AnimatedBackground from '../anim/AnimatedBackground';
import ChatWidget from '../chat/ChatWidget';

const TITLES = {
  '/dashboard': 'dashboard.title',
  '/patients': 'patients.title',
  '/consultation': 'consultation.title',
  '/medicines': 'medicines.title',
  '/rooms': 'rooms.title',
  '/printers': 'printers.title',
  '/surgeries': 'surgeries.title',
  '/imaging': 'imaging.title',
  '/rbac': 'rbac.title',
  '/operations': 'operations.title',
  '/appointments': 'appointments.title',
  '/beds': 'beds.title',
  '/billing': 'billing.title',
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
      {/* 院内即时通讯浮窗（全员可见，私聊 + 全院大群） */}
      <ChatWidget />
    </div>
  );
}
