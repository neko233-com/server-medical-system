import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Badge, Select, Textarea } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';

/**
 * Appointments — 预约挂号
 *
 * Covers the outpatient front-desk workflow:
 *   预约建档 (booked) → 到诊确认 (arrived) → 完成 (completed)
 * with cancelled / no_show as terminal side states.
 *
 * Every mutating action is gated by the `appointment:manage` permission so the
 * same page can be mounted for front-desk staff (full access) or for doctors
 * (read-only schedule view).
 */

// Maps the backend status to a badge tone.
const STATUS_TONE = {
  booked: 'info',
  arrived: 'amber',
  completed: 'green',
  cancelled: 'muted',
  no_show: 'red',
};

export default function Appointments() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const canManage = can('appointment:manage');

  const [items, setItems] = useState([]);
  const [patients, setPatients] = useState([]);
  const [doctors, setDoctors] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [detail, setDetail] = useState(null);
  const [error, setError] = useState('');
  const [form, setForm] = useState({
    patient_id: 0, doctor_id: 0, department: '', room_id: 0,
    slot_at: '', source: 'online', appt_type: 'general', reason: '', notes: '',
  });

  // Loads a page of appointments, optionally filtered by status.
  const load = useCallback(() => {
    setLoading(true);
    client
      .get('/appointments', { params: { page, page_size: 20, status: statusFilter || undefined } })
      .then((r) => {
        setItems(r.data.items || []);
        setTotal(r.data.pagination?.total || 0);
      })
      .catch(() => setError(t('common.load_failed')))
      .finally(() => setLoading(false));
  }, [page, statusFilter, t]);

  useEffect(() => { load(); }, [load]);

  // Reference data for the pickers.
  useEffect(() => {
    client.get('/patients', { params: { page_size: 200 } }).then((r) => setPatients(r.data.items || [])).catch(() => {});
    client.get('/rbac/directory').then((r) => setDoctors(r.data || [])).catch(() => {});
    client.get('/rooms', { params: { page_size: 100 } }).then((r) => setRooms(r.data.items || [])).catch(() => {});
  }, []);

  // Creates a new appointment from the modal form.
  const create = async () => {
    setError('');
    try {
      await client.post('/appointments', {
        patient_id: Number(form.patient_id),
        doctor_id: Number(form.doctor_id) || undefined,
        department: form.department || undefined,
        room_id: Number(form.room_id) || undefined,
        slot_at: form.slot_at ? new Date(form.slot_at).toISOString() : undefined,
        source: form.source,
        appt_type: form.appt_type,
        reason: form.reason,
        notes: form.notes,
      });
      setModal(false);
      setForm({ patient_id: 0, doctor_id: 0, department: '', room_id: 0, slot_at: '', source: 'online', appt_type: 'general', reason: '', notes: '' });
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Advances the appointment lifecycle.
  const setStatus = async (id, status) => {
    setError('');
    try {
      await client.put(`/appointments/${id}/status`, { status });
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Removes an appointment (front-desk only).
  const remove = async (id) => {
    setError('');
    try {
      await client.delete(`/appointments/${id}`);
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.delete_failed'));
    }
  };

  // Looks up a display name for a patient id.
  const pName = (id) => patients.find((p) => p.id === id)?.name || `#${id}`;

  const columns = [
    { key: 'appt_no', label: t('appointments.table.no'), render: (a) => <strong>{a.appt_no}</strong> },
    { key: 'patient_name', label: t('appointments.table.patient'), render: (a) => a.patient_name || pName(a.patient_id) },
    { key: 'doctor_name', label: t('appointments.table.doctor'), render: (a) => a.doctor_name || '—' },
    { key: 'department', label: t('appointments.table.department'), render: (a) => a.department || '—' },
    { key: 'slot_at', label: t('appointments.table.slot'), render: (a) => (a.slot_at ? new Date(a.slot_at).toLocaleString() : '—') },
    {
      key: 'status',
      label: t('appointments.table.status'),
      render: (a) => <Badge tone={STATUS_TONE[a.status] || 'neutral'}>{t(`appointments.status.${a.status}`)}</Badge>,
    },
    {
      key: 'ops',
      label: t('appointments.table.ops'),
      align: 'right',
      render: (a) => (
        <div className="btn-row">
          {canManage && a.status === 'booked' && (
            <Button size="sm" variant="secondary" onClick={() => setStatus(a.id, 'arrived')}>
              {t('appointments.action.arrive')}
            </Button>
          )}
          {canManage && a.status === 'arrived' && (
            <Button size="sm" variant="secondary" onClick={() => setStatus(a.id, 'completed')}>
              {t('appointments.action.complete')}
            </Button>
          )}
          {canManage && a.status === 'booked' && (
            <Button size="sm" variant="ghost" onClick={() => setStatus(a.id, 'cancelled')}>
              {t('appointments.action.cancel')}
            </Button>
          )}
          {canManage && a.status === 'booked' && (
            <Button size="sm" variant="danger" onClick={() => remove(a.id)}>
              {t('common.delete')}
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <FadeIn>
      <div className="toolbar">
        <Select value={statusFilter} onChange={(e) => { setPage(1); setStatusFilter(e.target.value); }} className="input--search">
          <option value="">{t('appointments.filter.all')}</option>
          {['booked', 'arrived', 'completed', 'cancelled', 'no_show'].map((s) => (
            <option key={s} value={s}>{t(`appointments.status.${s}`)}</option>
          ))}
        </Select>
        <span className="muted">{t('common.total')}: {total}</span>
        {canManage && (
          <Button icon="CalendarCheck" onClick={() => setModal(true)}>{t('appointments.new')}</Button>
        )}
      </div>

      {error && <div className="alert alert--error">{error}</div>}

      <Card padding={false}>
        {loading ? (
          <div className="center-pad"><Spinner size={26} /></div>
        ) : (
          <Table columns={columns} rows={items} empty={t('appointments.empty')} />
        )}
      </Card>

      {total > 20 && (
        <div className="pager">
          <Button variant="secondary" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>
            {t('common.prev')}
          </Button>
          <span className="muted">{page} / {Math.ceil(total / 20)}</span>
          <Button variant="secondary" size="sm" disabled={page >= Math.ceil(total / 20)} onClick={() => setPage(page + 1)}>
            {t('common.next')}
          </Button>
        </div>
      )}

      <Modal
        open={modal}
        onClose={() => setModal(false)}
        title={t('appointments.new')}
        footer={
          <>
            <Button variant="secondary" onClick={() => setModal(false)}>{t('common.cancel')}</Button>
            <Button onClick={create}>{t('common.save')}</Button>
          </>
        }
      >
        <Field label={t('appointments.patient')}>
          <Select value={form.patient_id} onChange={(e) => setForm({ ...form, patient_id: e.target.value })}>
            <option value="0">{t('appointments.patientPlaceholder')}</option>
            {patients.map((p) => <option key={p.id} value={p.id}>{p.name} ({p.code})</option>)}
          </Select>
        </Field>
        <Field label={t('appointments.doctor')}>
          <Select value={form.doctor_id} onChange={(e) => setForm({ ...form, doctor_id: e.target.value })}>
            <option value="0">{t('common.none')}</option>
            {doctors.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
          </Select>
        </Field>
        <Field label={t('appointments.department')}>
          <Input value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} placeholder={t('appointments.departmentPlaceholder')} />
        </Field>
        <Field label={t('appointments.room')}>
          <Select value={form.room_id} onChange={(e) => setForm({ ...form, room_id: e.target.value })}>
            <option value="0">{t('common.none')}</option>
            {rooms.map((r) => <option key={r.id} value={r.id}>{r.room_no} · {r.name_zh}</option>)}
          </Select>
        </Field>
        <Field label={t('appointments.slotAt')}>
          <Input type="datetime-local" value={form.slot_at} onChange={(e) => setForm({ ...form, slot_at: e.target.value })} />
        </Field>
        <Field label={t('appointments.source')}>
          <Select value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value })}>
            <option value="online">{t('appointments.source.online')}</option>
            <option value="phone">{t('appointments.source.phone')}</option>
            <option value="walkin">{t('appointments.source.walkin')}</option>
            <option value="referral">{t('appointments.source.referral')}</option>
          </Select>
        </Field>
        <Field label={t('appointments.type')}>
          <Select value={form.appt_type} onChange={(e) => setForm({ ...form, appt_type: e.target.value })}>
            <option value="general">{t('appointments.type.general')}</option>
            <option value="specialist">{t('appointments.type.specialist')}</option>
            <option value="followup">{t('appointments.type.followup')}</option>
          </Select>
        </Field>
        <Field label={t('appointments.reason')}>
          <Textarea rows={2} value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} />
        </Field>
      </Modal>
    </FadeIn>
  );
}
