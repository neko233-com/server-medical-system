import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Badge, Select, StatCard } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';

/**
 * Beds — 住院病区与床位管理
 *
 * Two levels:
 *   - Ward (病区): a physical inpatient area, e.g. 内科病区 / ICU
 *   - Bed  (床位): an individual bed inside a ward
 *
 * Bed lifecycle: available → reserved → occupied → available,
 * with `maintenance` as an offline state.
 * Admits / discharges are recorded with a timestamp for LOS analytics.
 */

// Bed status → badge tone.
const BED_TONE = {
  available: 'green',
  occupied: 'red',
  reserved: 'amber',
  maintenance: 'muted',
};

export default function Beds() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const canManage = can('bed:manage');

  const [wards, setWards] = useState([]);
  const [beds, setBeds] = useState([]);
  const [patients, setPatients] = useState([]);
  const [total, setTotal] = useState(0);
  const [wardFilter, setWardFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [wardModal, setWardModal] = useState(false);
  const [bedModal, setBedModal] = useState(false);
  const [admitModal, setAdmitModal] = useState(false);
  const [admitTarget, setAdmitTarget] = useState(null);
  const [admitPatient, setAdmitPatient] = useState(0);
  const [error, setError] = useState('');
  const [wardForm, setWardForm] = useState({ code: '', name_zh: '', name_en: '', department: '', floor: '' });
  const [bedForm, setBedForm] = useState({ ward_id: 0, bed_no: '', status: 'available', daily_rate: 80 });

  // Loads wards and beds.
  const load = useCallback(() => {
    setLoading(true);
    Promise.all([
      client.get('/wards'),
      client.get('/beds', { params: { ward_id: wardFilter || undefined, status: statusFilter || undefined, page_size: 200 } }),
    ])
      .then(([w, b]) => {
        setWards(w.data.items || []);
        setBeds(b.data.items || []);
        setTotal(b.data.pagination?.total || 0);
      })
      .catch(() => setError(t('common.load_failed')))
      .finally(() => setLoading(false));
  }, [wardFilter, statusFilter, t]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    client.get('/patients', { params: { page_size: 200 } }).then((r) => setPatients(r.data.items || [])).catch(() => {});
  }, []);

  // Creates a ward.
  const createWard = async () => {
    setError('');
    try {
      await client.post('/wards', wardForm);
      setWardModal(false);
      setWardForm({ code: '', name_zh: '', name_en: '', department: '', floor: '' });
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Creates a bed inside a ward.
  const createBed = async () => {
    setError('');
    try {
      await client.post('/beds', { ...bedForm, ward_id: Number(bedForm.ward_id), daily_rate: Number(bedForm.daily_rate) || 0 });
      setBedModal(false);
      setBedForm({ ward_id: 0, bed_no: '', status: 'available', daily_rate: 80 });
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Deletes a bed.
  const removeBed = async (id) => {
    setError('');
    try {
      await client.delete(`/beds/${id}`);
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.delete_failed'));
    }
  };

  // Admits a patient into a bed.
  const admit = async () => {
    setError('');
    try {
      await client.put(`/beds/${admitTarget.id}/admit`, { patient_id: Number(admitPatient) });
      setAdmitModal(false);
      setAdmitPatient(0);
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Discharges the occupant and frees the bed.
  const discharge = async (id) => {
    setError('');
    try {
      await client.put(`/beds/${id}/discharge`);
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Client-side occupancy summary for the KPI strip.
  const stat = {
    total: beds.length,
    occupied: beds.filter((b) => b.status === 'occupied').length,
    available: beds.filter((b) => b.status === 'available').length,
    maintenance: beds.filter((b) => b.status === 'maintenance').length,
  };
  const rate = stat.total ? Math.round((stat.occupied / stat.total) * 1000) / 10 : 0;

  const columns = [
    { key: 'code', label: t('beds.table.code'), render: (b) => <strong>{b.code || b.bed_no}</strong> },
    { key: 'ward_name', label: t('beds.table.ward'), render: (b) => b.ward_name || '—' },
    { key: 'bed_no', label: t('beds.table.bedNo'), render: (b) => b.bed_no },
    {
      key: 'status',
      label: t('beds.table.status'),
      render: (b) => <Badge tone={BED_TONE[b.status] || 'neutral'}>{t(`beds.status.${b.status}`)}</Badge>,
    },
    { key: 'patient_name', label: t('beds.table.patient'), render: (b) => b.patient_name || '—' },
    {
      key: 'admitted_at',
      label: t('beds.table.admittedAt'),
      render: (b) => (b.admitted_at ? new Date(b.admitted_at).toLocaleString() : '—'),
    },
    {
      key: 'daily_rate',
      label: t('beds.table.dailyRate'),
      align: 'right',
      render: (b) => `¥${Number(b.daily_rate || 0).toFixed(2)}`,
    },
    {
      key: 'ops',
      label: t('common.actions'),
      align: 'right',
      render: (b) => (
        <div className="btn-row">
          {canManage && b.status !== 'occupied' && (
            <Button size="sm" variant="secondary" icon="UserCheck"
              onClick={() => { setAdmitTarget(b); setAdmitPatient(0); setAdmitModal(true); }}>
              {t('beds.action.admit')}
            </Button>
          )}
          {canManage && b.status === 'occupied' && (
            <Button size="sm" variant="secondary" icon="Logout" onClick={() => discharge(b.id)}>
              {t('beds.action.discharge')}
            </Button>
          )}
          {canManage && (
            <Button size="sm" variant="danger" icon="Trash" onClick={() => removeBed(b.id)} />
          )}
        </div>
      ),
    },
  ];

  return (
    <FadeIn>
      <div className="stat-grid">
        <StatCard label={t('beds.stat.total')} value={stat.total} icon="Armchair" tone="primary" />
        <StatCard label={t('beds.stat.occupied')} value={stat.occupied} icon="UserCheck" tone="red" />
        <StatCard label={t('beds.stat.available')} value={stat.available} icon="Armchair" tone="green" />
        <StatCard label={t('beds.stat.occupancy')} value={`${rate}%`} icon="Layers" tone="amber" hint={t('beds.stat.occupancyHint')} />
      </div>

      {error && <div className="alert alert--error">{error}</div>}

      <div className="toolbar">
        <Select value={wardFilter} onChange={(e) => setWardFilter(e.target.value)}>
          <option value="">{t('beds.filter.allWards')}</option>
          {wards.map((w) => <option key={w.id} value={w.id}>{w.name_zh}</option>)}
        </Select>
        <Select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="">{t('beds.filter.allStatus')}</option>
          {['available', 'occupied', 'reserved', 'maintenance'].map((s) => (
            <option key={s} value={s}>{t(`beds.status.${s}`)}</option>
          ))}
        </Select>
        <span className="muted">{t('common.total')}: {total}</span>
        {canManage && (
          <>
            <Button icon="Building2" onClick={() => setWardModal(true)}>{t('beds.newWard')}</Button>
            <Button icon="Plus" onClick={() => setBedModal(true)}>{t('beds.newBed')}</Button>
          </>
        )}
      </div>

      <Card padding={false}>
        {loading ? (
          <div className="center-pad"><Spinner size={26} /></div>
        ) : (
          <Table columns={columns} rows={beds} empty={t('beds.empty')} />
        )}
      </Card>

      {/* ---- 新建病区 ---- */}
      <Modal open={wardModal} onClose={() => setWardModal(false)} title={t('beds.newWard')} footer={
        <>
          <Button variant="ghost" onClick={() => setWardModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={createWard}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('beds.wardCode')}><Input value={wardForm.code} onChange={(e) => setWardForm({ ...wardForm, code: e.target.value })} placeholder="W-IN" /></Field>
          <Field label={t('beds.wardName')}>
            <Input value={wardForm.name_zh} onChange={(e) => setWardForm({ ...wardForm, name_zh: e.target.value })} />
          </Field>
          <Field label={t('beds.wardNameEn')}>
            <Input value={wardForm.name_en} onChange={(e) => setWardForm({ ...wardForm, name_en: e.target.value })} />
          </Field>
          <Field label={t('beds.department')}>
            <Input value={wardForm.department} onChange={(e) => setWardForm({ ...wardForm, department: e.target.value })} />
          </Field>
          <Field label={t('beds.floor')}>
            <Input value={wardForm.floor} onChange={(e) => setWardForm({ ...wardForm, floor: e.target.value })} placeholder="5F" />
          </Field>
        </div>
      </Modal>

      {/* ---- 新建床位 ---- */}
      <Modal open={bedModal} onClose={() => setBedModal(false)} title={t('beds.newBed')} footer={
        <>
          <Button variant="ghost" onClick={() => setBedModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={createBed}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('beds.ward')}>
            <Select value={bedForm.ward_id} onChange={(e) => setBedForm({ ...bedForm, ward_id: e.target.value })}>
              <option value={0}>{t('common.select')}</option>
              {wards.map((w) => <option key={w.id} value={w.id}>{w.name_zh}</option>)}
            </Select>
          </Field>
          <Field label={t('beds.bedNo')}>
            <Input value={bedForm.bed_no} onChange={(e) => setBedForm({ ...bedForm, bed_no: e.target.value })} placeholder="01" />
          </Field>
          <Field label={t('beds.status')}>
            <Select value={bedForm.status} onChange={(e) => setBedForm({ ...bedForm, status: e.target.value })}>
              {['available', 'reserved', 'maintenance'].map((s) => (
                <option key={s} value={s}>{t(`beds.status.${s}`)}</option>
              ))}
            </Select>
          </Field>
          <Field label={t('beds.dailyRate')}>
            <Input type="number" min="0" step="0.01" value={bedForm.daily_rate}
              onChange={(e) => setBedForm({ ...bedForm, daily_rate: e.target.value })} />
          </Field>
        </div>
      </Modal>

      {/* ---- 入院登记 ---- */}
      <Modal open={admitModal} onClose={() => setAdmitModal(false)} title={t('beds.action.admit')} footer={
        <>
          <Button variant="ghost" onClick={() => setAdmitModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={admit}>{t('beds.action.admit')}</Button>
        </>
      }>
        {admitTarget && (
          <p className="muted">
            {t('beds.bedLabel')}: <strong>{admitTarget.code || admitTarget.bed_no}</strong>
          </p>
        )}
        <Field label={t('appointments.patient')}>
          <Select value={admitPatient} onChange={(e) => setAdmitPatient(e.target.value)}>
            <option value={0}>{t('appointments.patientPlaceholder')}</option>
            {patients.map((p) => <option key={p.id} value={p.id}>{p.name} ({p.code})</option>)}
          </Select>
        </Field>
      </Modal>
    </FadeIn>
  );
}
