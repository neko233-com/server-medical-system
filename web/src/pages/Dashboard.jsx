import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, StatCard, Button, Spinner, Empty } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';
import Reicon from '../components/Reicon';
import { genderLabel } from '../utils/format';

// The integrated hospital-operations modules. Each is permission-gated so a user
// only sees the stat card / shortcut for modules they are allowed to use.
const MODULES = [
  { key: 'rooms', perm: 'room:read', route: '/rooms', icon: 'Building', tone: 'primary' },
  { key: 'printers', perm: 'printer:read', route: '/printers', icon: 'Printer', tone: 'teal' },
  { key: 'surgeries', perm: 'surgery:read', route: '/surgeries', icon: 'Scissors', tone: 'violet' },
  { key: 'imaging', perm: 'imaging:read', route: '/imaging', icon: 'Image', tone: 'amber' },
  { key: 'appointments', perm: 'appointment:read', route: '/appointments', icon: 'CalendarCheck', tone: 'primary' },
  { key: 'beds', perm: 'bed:read', route: '/beds', icon: 'Armchair', tone: 'green' },
  { key: 'billing', perm: 'billing:read', route: '/billing', icon: 'Receipt', tone: 'teal' },
];

export default function Dashboard() {
  const { t } = useTranslation();
  const { user, can } = useAuth();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    patients: 0,
    medicines: 0,
    prescriptions: 0,
    casesToday: 0,
    rooms: 0,
    printers: 0,
    surgeries: 0,
    imaging: 0,
    appointments: 0,
    beds: 0,
    billing: 0,
  });
  const [recent, setRecent] = useState([]);

  useEffect(() => {
    // allSettled: a user lacking permission for a module simply gets 0 for it
    // instead of the whole dashboard failing to load.
    Promise.allSettled([
      client.get('/patients?page_size=5'),
      client.get('/medicines?page_size=1'),
      client.get('/prescriptions?page_size=1'),
      client.get('/cases?page_size=50'),
      client.get('/rooms?page_size=1'),
      client.get('/printers?page_size=1'),
      client.get('/surgeries?page_size=1'),
      client.get('/imaging?page_size=1'),
      client.get('/appointments?page_size=1'),
      client.get('/beds?page_size=1'),
      client.get('/invoices?page_size=1'),
    ])
      .then(([p, m, rx, cs, rm, pr, sg, im, ap, bd, bl]) => {
        const total = (res, d = 0) =>
          res.status === 'fulfilled' && res.value?.data?.pagination ? res.value.data.pagination.total : d;
        const today = new Date().toISOString().slice(0, 10);
        const casesToday = cs.status === 'fulfilled'
          ? (cs.value?.data?.items || []).filter((c) => (c.visit_date || '').slice(0, 10) === today).length
          : 0;
        setStats({
          patients: total(p),
          medicines: total(m),
          prescriptions: total(rx),
          casesToday,
          rooms: total(rm),
          printers: total(pr),
          surgeries: total(sg),
          imaging: total(im),
          appointments: total(ap),
          beds: total(bd),
          billing: total(bl),
        });
        setRecent(cs.status === 'fulfilled' ? (cs.value?.data?.items || []) : []);
      })
      .finally(() => setLoading(false));
  }, []);

  const visibleModules = MODULES.filter((m) => can(m.perm));
  const canRbac = can('user:read') || can('role:read') || can('user:manage') || can('role:manage');
  const canOps = can('report:read');

  return (
    <div className="stack-lg">
      <FadeIn>
        <div className="dash-greet">
          <h1>{t('login.welcome')}，{user?.name}</h1>
          <p>{t('app.tagline')}</p>
        </div>
      </FadeIn>

      <FadeIn delay={60}>
        <div className="stat-grid">
          <StatCard tone="primary" icon="Users" label={t('dashboard.statPatients')} value={loading ? '—' : stats.patients} />
          <StatCard tone="teal" icon="Pill" label={t('dashboard.statMedicines')} value={loading ? '—' : stats.medicines} />
          <StatCard tone="violet" icon="Stethoscope" label={t('dashboard.statCases')} value={loading ? '—' : stats.casesToday} />
          <StatCard tone="amber" icon="Notebook" label={t('dashboard.statPrescriptions')} value={loading ? '—' : stats.prescriptions} />
          {visibleModules.map((m) => (
            <StatCard
              key={m.key}
              tone={m.tone}
              icon={m.icon}
              label={t(`dashboard.stat${m.key.charAt(0).toUpperCase() + m.key.slice(1)}`)}
              value={loading ? '—' : stats[m.key]}
              onClick={() => navigate(m.route)}
            />
          ))}
        </div>
      </FadeIn>

      {visibleModules.length > 0 && (
        <FadeIn delay={100}>
          <Card title={t('dashboard.quickActions')}>
            <div className="quick-grid">
              {visibleModules.map((m) => (
                <button key={m.key} className="quick-card" onClick={() => navigate(m.route)}>
                  <span className={`quick-card__icon quick-card__icon--${m.tone}`}>
                    <Reicon name={m.icon} size={22} />
                  </span>
                  <span className="quick-card__label">{t(`dashboard.quick.${m.key}`)}</span>
                  <Reicon name="ChevronRight" size={18} color="var(--muted)" />
                </button>
              ))}
              {canRbac && (
                <button className="quick-card" onClick={() => navigate('/rbac')}>
                  <span className="quick-card__icon quick-card__icon--rose">
                    <Reicon name="Shield" size={22} />
                  </span>
                  <span className="quick-card__label">{t('dashboard.quick.rbac')}</span>
                  <Reicon name="ChevronRight" size={18} color="var(--muted)" />
                </button>
              )}
              {canOps && (
                <button className="quick-card" onClick={() => navigate('/operations')}>
                  <span className="quick-card__icon quick-card__icon--primary">
                    <Reicon name="Activity" size={22} />
                  </span>
                  <span className="quick-card__label">{t('operations.title')}</span>
                  <Reicon name="ChevronRight" size={18} color="var(--muted)" />
                </button>
              )}
            </div>
          </Card>
        </FadeIn>
      )}

      <FadeIn delay={120}>
        <Card
          title={t('dashboard.recentPatients')}
          actions={<Button variant="ghost" size="sm" icon="ArrowRight" onClick={() => navigate('/patients')}>{t('dashboard.viewAll')}</Button>}
        >
          {loading ? (
            <div className="center-pad"><Spinner /></div>
          ) : recent.length === 0 ? (
            <Empty icon="Users" text={t('dashboard.noRecent')} />
          ) : (
            <div className="recent-list">
              {recent.map((p) => (
                <button key={p.id} className="recent-item" onClick={() => navigate(`/patients/${p.id}`)}>
                  <span className="avatar avatar--sm">{(p.name || '?').slice(0, 1)}</span>
                  <div className="recent-item__meta">
                    <strong>{p.name}</strong>
                    <small>{genderLabel(p.gender, t)} · {p.phone || '—'}</small>
                  </div>
                  <Reicon name="ChevronRight" size={18} color="var(--muted)" />
                </button>
              ))}
            </div>
          )}
        </Card>
      </FadeIn>
    </div>
  );
}
