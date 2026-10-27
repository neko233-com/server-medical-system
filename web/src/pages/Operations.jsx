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
  const [revenue, setRevenue] = useState(null);

  useEffect(() => {
    setLoading(true);
    Promise.allSettled([
      client.get('/operations/overview'),
      client.get('/operations/trend', { params: { days } }),
      client.get('/operations/departments'),
      client.get('/operations/doctors'),
      client.get('/operations/status'),
      client.get('/operations/revenue', { params: { days } }),
    ]).then(([o, tr, dp, dc, st, rv]) => {
      if (o.status === 'fulfilled') setOv(o.value.data);
      if (tr.status === 'fulfilled') setTrend(tr.value.data.series || []);
      if (dp.status === 'fulfilled') setDepts(dp.value.data.items || []);
      if (dc.status === 'fulfilled') setDoctors(dc.value.data.items || []);
      if (st.status === 'fulfilled') setStatus(st.value.data);
      if (rv.status === 'fulfilled') setRevenue(rv.value.data);
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
          <FadeIn delay={95}>
            <div className="stat-grid">
              <StatCard tone="green" icon="Receipt" label={t('operations.paidMonth')} value={formatMoney(ov.paid_month)} hint={t('operations.netRevenue')} />
              <StatCard tone="primary" icon="CalendarCheck" label={t('operations.appointmentsToday')} value={ov.appointments_today} hint={t('operations.appointmentsPending') + ': ' + ov.appointments_pending} />
              <StatCard tone="violet" icon="Armchair" label={t('operations.bedOccupancy')} value={`${ov.bed_occupancy}%`} hint={`${ov.beds_occupied}/${ov.beds_total}`} />
              <StatCard tone="amber" icon="Wallet" label={t('operations.invoicesPending')} value={ov.invoices_pending} />
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

          {/* 住院床位占用 */}
          {status && (status.beds || []).length > 0 && (
            <FadeIn delay={220}>
              <Card title={t('operations.bedStatus')}>
                <Table
                  rowKey="ward_id"
                  columns={[
                    { key: 'ward_name', label: t('operations.ward') },
                    { key: 'total', label: t('operations.bedTotal') },
                    { key: 'occupied', label: t('operations.bedOccupied') },
                    {
                      key: 'occupancy',
                      label: t('operations.bedOccupancy'),
                      render: (r) => (
                        <Badge tone={r.occupancy >= 90 ? 'red' : r.occupancy >= 60 ? 'amber' : 'green'}>
                          {r.occupancy}%
                        </Badge>
                      ),
                    },
                  ]}
                  rows={status.beds || []}
                  empty={t('common.empty')}
                />
              </Card>
            </FadeIn>
          )}

          {/* 收入构成分析 */}
          {revenue && (
            <FadeIn delay={240}>
              <Card title={t('operations.revenueBreakdown')}>
                <div className="stat-grid" style={{ marginBottom: 16 }}>
                  <StatCard tone="primary" icon="Receipt" label={t('operations.totalBilled')} value={formatMoney(revenue.total_billed)} />
                  <StatCard tone="green" icon="Dollar" label={t('operations.totalPaid')} value={formatMoney(revenue.total_paid)} />
                  <StatCard tone="red" icon="Undo" label={t('operations.totalRefund')} value={formatMoney(revenue.total_refund)} />
                </div>

                <div className="grid-2">
                  <div>
                    <h4 className="sub-head">{t('operations.byMethod')}</h4>
                    <Table
                      rowKey="method"
                      columns={[
                        { key: 'method', label: t('operations.method'), render: (r) => t(`billing.method.${r.method}`) },
                        { key: 'count', label: t('operations.count') },
                        { key: 'amount', label: t('operations.amount'), render: (r) => formatMoney(r.amount) },
                      ]}
                      rows={revenue.by_method || []}
                      empty={t('common.empty')}
                    />
                  </div>
                  <div>
                    <h4 className="sub-head">{t('operations.byItem')}</h4>
                    <Table
                      rowKey="item_type"
                      columns={[
                        { key: 'item_type', label: t('operations.itemType'), render: (r) => t(`billing.itemType.${r.item_type}`) },
                        { key: 'count', label: t('operations.count') },
                        { key: 'amount', label: t('operations.amount'), render: (r) => formatMoney(r.amount) },
                      ]}
                      rows={revenue.by_item || []}
                      empty={t('common.empty')}
                    />
                  </div>
                </div>
              </Card>
            </FadeIn>
          )}
        </>
      )}
    </div>
  );
}
