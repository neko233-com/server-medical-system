import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import Reicon from '../Reicon';
import LanguageSwitcher from '../LanguageSwitcher';

export default function Topbar({ title }) {
  const { t } = useTranslation();
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const onLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <header className="topbar">
      <h2 className="topbar__title">{title}</h2>
      <div className="topbar__right">
        <LanguageSwitcher />
        <div className="topbar__user">
          <span className="avatar">{(user?.name || 'U').slice(0, 1)}</span>
          <div className="topbar__user-meta">
            <strong>{user?.name}</strong>
            <small>{user?.role === 'admin' ? 'Admin' : 'Doctor'}</small>
          </div>
          <button className="icon-btn" title={t('common.logout')} onClick={onLogout}>
            <Reicon name="Logout" size={18} />
          </button>
        </div>
      </div>
    </header>
  );
}
