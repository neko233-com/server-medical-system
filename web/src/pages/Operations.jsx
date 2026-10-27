import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, StatCard, Table, Badge, Spinner, Button } from '../components/ui';
import FadeIn from '../components/anim/FadeIn';
import { formatMoney } from '../utils/format';

// 医院运营总览（管理员后台运营）
// 数据来源：/api/operations/* ，全部只读聚合，按 report:read 权限受控。
export default function Operations() {
  const { t } = useTranslation();
  const [days, setDays] = useState(14);
  const [loading, setLoading] = useState(true);
  const [ov, setOv] = useState(null);
  const [trend, setTrend] = useState([]);
  const [depts, setDepts] = useState([]);
  const [doctors, setDoctors] = useState([]);
  const [status, setStatus] = useState(null);

  useEffect(() => {
    setLoading(true);
    Promise.allSettled([
      client.get('/operations/overview'),
      client.get('/operations/trend', { params: { days } }),
      client.get('/operations/departments'),
      client.get('/operations/doctors'),
      client.get('/operations/status'),
    ]).then(([o, tr, dp, dc, st]) => {
      if (o.status === 'fulfilled') setOv(o.value.data);
      if (tr.status === 'fulfilled') setTrend(tr.value.data.series || []);
      if (dp.status === 'fulfilled') setDepts(dp.value.data.items || []);
      if (dc.status === 'fulfilled') setDoctors(dc.value.data.items || []);
      if (st.status === 'fulfilled') setStatus(st.value.data);
      setLoading(false);
    });
  }, [days]);

  const maxVisits = Math.max(1, ...trend.map((d) => d.visits));

  return (
    <div className="stack-lg">
      <FadeIn>
        <div className="row-between">
          <div>
            <h2 style={{ margin: 0 }}>{t('operations.title')}</h2>
            <p style={{ margin: '4px 0 0', color: 'var(--muted)' }}>{t('operations.subtitle')}</p>
          </div>
          <div className="btn-row">
            {[7, 14, 30].map((d) => (
              <Button key={d} size="sm" variant={days === d ? 'primary' : 'ghost'} onClick={() => setDays(d)}>
                {d}{t('operations.days')}
              </Button>
            ))}
          </div>
        </div>
      </FadeIn>

      {loading || !ov ? (
        <div className="center-pad"><Spinner /></div>
      ) : (
        <>
          {/* 核心 KPI */}
          <FadeIn delay={50}>
            <div className="stat-grid">
              <StatCard tone="primary" icon="Users" label={t('operations.patients')} value={ov.patients} />
              <StatCard tone="teal" icon="Stethoscope" label={t('operations.visitsToday')} value={ov.visits_today} />
              <StatCard tone="violet" icon="Notebook" label={t('operations.prescToday')} value={ov.prescriptions_today} />
              <StatCard tone="amber" icon="Dollar" label={t('operations.revenueToday')} value={formatMoney(ov.revenue_today)} />
            </div>
          </FadeIn>
          <FadeIn delay={80}>
            <div className="stat-grid">
              <StatCard tone="primary" icon="Dollar2" label={t('operations.revenueMonth')} value={formatMoney(ov.revenue_month)} />
              <StatCard tone="violet" icon="Scissors" label={t('operations.pendingSurgeries')} value={ov.pending_surgeries} />
              <StatCard tone="amber" icon="Image" label={t('operations.pendingImaging')} value={ov.pending_imaging} />
              <StatCard tone="teal" icon="Box" label={t('operations.lowStock')} value={ov.low_stock} />
            </div>
          </FadeIn>

          {/* 门诊量趋势（纯 CSS 条形图，无第三方依赖） */}
          <FadeIn delay={110}>
            <Card title={t('operations.trend')}>
              <div className="trend-chart">
                {trend.map((d) => (
                  <div key={d.date} className="trend-col" title={`${d.date} · ${t('operations.visits')}: ${d.visits} · ${formatMoney(d.revenue)}`}>
                    <div className="trend-bar" style={{ height: `${(d.visits / maxVisits) * 100}%` }} />
                    <span className="trend-date">{d.date.slice(5)}</span>
                  </div>
                ))}
              </div>
            </Card>
          </FadeIn>

          {/* 科室效能 */}
          <FadeIn delay={140}>
            <Card title={t('operations.departments')}>
              <Table
                rowKey="department"
                columns={[
                  { key: 'department', label: t('operations.department') },
                  { key: 'doctors', label: t('operations.doctors') },
                  { key: 'visits', label: t('operations.visits') },
                  { key: 'revenue', label: t('operations.revenue'), render: (r) => formatMoney(r.revenue) },
                ]}
                rows={depts}
                empty={t('common.empty')}
              />
            </Card>
          </FadeIn>

          {/* 医生效能 */}
          <FadeIn delay={170}>
            <Card title={t('operations.doctorPerf')}>
              <Table
                columns={[
                  { key: 'name', label: t('operations.doctor') },
                  { key: 'department', label: t('operations.department') },
                  { key: 'visits', label: t('operations.visits') },
                  { key: 'prescriptions', label: t('operations.prescriptions') },
                  { key: 'revenue', label: t('operations.revenue'), render: (r) => formatMoney(r.revenue) },
                ]}
                rows={doctors}
                empty={t('common.empty')}
              />
            </Card>
          </FadeIn>

          {/* 资源实时状态 */}
          {status && (
            <FadeIn delay={200}>
              <Card title={t('operations.resources')}>
                <div className="quick-grid">
                  {(status.rooms || []).map((r) => (
                    <div key={r.id} className="quick-card" style={{ cursor: 'default' }}>
                      <span className={`quick-card__icon quick-card__icon--${r.occupied ? 'amber' : 'teal'}`}>
                        {r.occupied ? '●' : '○'}
                      </span>
                      <span className="quick-card__label">
                        {r.room_no} · {r.name_zh}
                        <br />
                        <small style={{ color: 'var(--muted)' }}>
                          {t('operations.visitsToday')}: {r.visits_today}
                        </small>
                      </span>
                      <Badge tone={r.occupied ? 'amber' : 'green'}>
                        {r.occupied ? t('operations.busy') : t('operations.idle')}
                      </Badge>
                    </div>
                  ))}
                </div>
              </Card>
            </FadeIn>
          )}
        </>
      )}
    </div>
  );
}
