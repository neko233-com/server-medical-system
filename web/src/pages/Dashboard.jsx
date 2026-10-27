import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, StatCard, Button, Spinner, Empty } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';
import Reicon from '../components/Reicon';
import { genderLabel } from '../utils/format';

export default function Dashboard() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({ patients: 0, medicines: 0, prescriptions: 0, casesToday: 0 });
  const [recent, setRecent] = useState([]);

  useEffect(() => {
    Promise.all([
      client.get('/patients?page_size=5'),
      client.get('/medicines?page_size=1'),
      client.get('/prescriptions?page_size=1'),
      client.get('/cases?page_size=50'),
    ])
      .then(([p, m, rx, cs]) => {
        const today = new Date().toISOString().slice(0, 10);
        const casesToday = (cs.data.items || []).filter(
          (c) => (c.visit_date || '').slice(0, 10) === today
        ).length;
        setStats({
          patients: p.data.pagination.total,
          medicines: m.data.pagination.total,
          prescriptions: rx.data.pagination.total,
          casesToday,
        });
        setRecent(p.data.items || []);
      })
      .finally(() => setLoading(false));
  }, []);

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
        </div>
      </FadeIn>

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
