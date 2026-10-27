import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Badge, Select, Textarea } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';

const STATUS_TONE = { planned: 'info', in_progress: 'warning', done: 'success', cancelled: 'muted' };

export default function Surgeries() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const [items, setItems] = useState([]);
  const [patients, setPatients] = useState([]);
  const [doctors, setDoctors] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [detail, setDetail] = useState(null);
  const [form, setForm] = useState({
    patient_id: 0, surgeon_id: 0, anesthetist: '', room_id: 0, operation_name: '', operation_name_en: '',
    scheduled_at: '', anesthesia_type: '', pre_op_diagnosis: '', notes: '',
  });

  const load = useCallback(() => {
    setLoading(true);
    client.get('/surgeries', { params: { q, page, page_size: 10 } }).then((r) => {
      setItems(r.data.items); setTotal(r.data.pagination.total);
    }).finally(() => setLoading(false));
  }, [q, page]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    client.get('/patients', { params: { page_size: 200 } }).then((r) => setPatients(r.data.items)).catch(() => {});
    client.get('/rbac/directory').then((r) => setDoctors(r.data)).catch(() => {});
    client.get('/rooms', { params: { page_size: 100 } }).then((r) => setRooms(r.data.items)).catch(() => {});
  }, []);

  const create = async () => {
    const payload = {
      ...form,
      patient_id: Number(form.patient_id),
      surgeon_id: Number(form.surgeon_id) || undefined,
      room_id: Number(form.room_id) || undefined,
      scheduled_at: form.scheduled_at ? new Date(form.scheduled_at).toISOString() : undefined,
    };
    await client.post('/surgeries', payload);
    setModal(false);
    setForm({ patient_id: 0, surgeon_id: 0, anesthetist: '', room_id: 0, operation_name: '', operation_name_en: '', scheduled_at: '', anesthesia_type: '', pre_op_diagnosis: '', notes: '' });
    load();
  };

  const openDetail = (s) => { setDetail(s); };

  const setStatus = async (status) => {
    const payload = { status };
    if (status === 'done') {
      payload.findings = detail.findings || '';
      payload.procedure_notes = detail.procedure_notes || '';
      payload.complications = detail.complications || '';
      payload.post_op_diagnosis = detail.post_op_diagnosis || '';
      payload.duration_min = Number(detail.duration_min) || 0;
    }
    const { data } = await client.put(`/surgeries/${detail.id}/status`, payload);
    setDetail(data.item);
    load();
  };

  const updateReport = async () => {
    const { data } = await client.put(`/surgeries/${detail.id}`, {
      operation_name: detail.operation_name,
      operation_name_en: detail.operation_name_en,
      anesthesia_type: detail.anesthesia_type,
      pre_op_diagnosis: detail.pre_op_diagnosis,
      notes: detail.notes,
      findings: detail.findings, procedure_notes: detail.procedure_notes,
      complications: detail.complications, post_op_diagnosis: detail.post_op_diagnosis,
    });
    setDetail(data.item);
  };

  const pName = (id) => patients.find((p) => p.id === id)?.name || `#${id}`;
  const dName = (id) => doctors.find((d) => d.id === id)?.name || (id ? `#${id}` : '—');
  const rNo = (id) => rooms.find((r) => r.id === id)?.room_no || (id ? `#${id}` : '—');

  const columns = [
    { key: 'surgery_no', label: t('surgeries.table.no'), render: (s) => <strong>{s.surgery_no}</strong> },
    { key: 'patient', label: t('surgeries.table.patient'), render: (s) => pName(s.patient_id) },
    { key: 'operation', label: t('surgeries.table.operation'), render: (s) => s.operation_name },
    { key: 'surgeon', label: t('surgeries.table.surgeon'), render: (s) => s.surgeon_name || dName(s.surgeon_id) },
    { key: 'room', label: t('surgeries.table.room'), render: (s) => rNo(s.room_id) },
    { key: 'scheduled', label: t('surgeries.table.scheduled'), render: (s) => (s.scheduled_at || '').slice(0, 16).replace('T', ' ') },
    { key: 'status', label: t('surgeries.table.status'), render: (s) => <Badge tone={STATUS_TONE[s.status]}>{t('surgeries.status.' + s.status)}</Badge> },
    {
      key: 'actions', label: t('common.actions'), align: 'right',
      render: (s) => <Button size="sm" variant="ghost" icon="Eye" onClick={() => openDetail(s)}>{t('common.detail')}</Button>,
    },
  ];

  return (
    <FadeIn>
      <div className="toolbar">
        <Input className="input--search" placeholder={t('surgeries.search')} value={q} onChange={(e) => { setQ(e.target.value); setPage(1); }} />
        {can('surgery:manage') && <Button icon="Plus" onClick={() => setModal(true)}>{t('surgeries.new')}</Button>}
      </div>

      <Card padding={false}>
        {loading ? <div className="center-pad"><Spinner /></div> : <Table columns={columns} rows={items} empty={t('common.empty')} />}
      </Card>

      <Modal open={modal} onClose={() => setModal(false)} title={t('surgeries.new')} footer={
        <>
          <Button variant="ghost" onClick={() => setModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={create}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('surgeries.patient')} className="span-2">
            <Select value={form.patient_id} onChange={(e) => setForm({ ...form, patient_id: Number(e.target.value) })}>
              <option value={0}>—</option>{patients.map((p) => <option key={p.id} value={p.id}>{p.name}（{p.gender}）</option>)}
            </Select>
          </Field>
          <Field label={t('surgeries.surgeon')}>
            <Select value={form.surgeon_id} onChange={(e) => setForm({ ...form, surgeon_id: Number(e.target.value) })}>
              <option value={0}>—</option>{doctors.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
            </Select>
          </Field>
          <Field label={t('surgeries.room')}>
            <Select value={form.room_id} onChange={(e) => setForm({ ...form, room_id: Number(e.target.value) })}>
              <option value={0}>—</option>{rooms.filter((r) => r.type === 'surgery').map((r) => <option key={r.id} value={r.id}>{r.room_no} {r.name_zh}</option>)}
            </Select>
          </Field>
          <Field label={t('surgeries.operationName')} className="span-2"><Input value={form.operation_name} onChange={(e) => setForm({ ...form, operation_name: e.target.value })} /></Field>
          <Field label={t('surgeries.operationNameEn')} className="span-2"><Input value={form.operation_name_en} onChange={(e) => setForm({ ...form, operation_name_en: e.target.value })} /></Field>
          <Field label={t('surgeries.scheduledAt')}><Input type="datetime-local" value={form.scheduled_at} onChange={(e) => setForm({ ...form, scheduled_at: e.target.value })} /></Field>
          <Field label={t('surgeries.anesthesiaType')}><Input value={form.anesthesia_type} onChange={(e) => setForm({ ...form, anesthesia_type: e.target.value })} placeholder="全麻/椎管内" /></Field>
          <Field label={t('surgeries.anesthetist')}><Input value={form.anesthetist} onChange={(e) => setForm({ ...form, anesthetist: e.target.value })} /></Field>
          <Field label={t('surgeries.preOpDiagnosis')} className="span-2"><Input value={form.pre_op_diagnosis} onChange={(e) => setForm({ ...form, pre_op_diagnosis: e.target.value })} /></Field>
          <Field label={t('surgeries.notes')} className="span-2"><Textarea value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
        </div>
      </Modal>

      <Modal open={!!detail} onClose={() => setDetail(null)} width={680} title={`${t('surgeries.title')} · ${detail?.surgery_no}`} footer={
        <div className="row-actions" style={{ justifyContent: 'flex-end' }}>
          {detail && detail.status === 'planned' && can('surgery:manage') && <Button icon="Play" onClick={() => setStatus('in_progress')}>{t('surgeries.start')}</Button>}
          {detail && detail.status === 'in_progress' && can('surgery:manage') && <Button icon="Check" onClick={() => setStatus('done')}>{t('surgeries.complete')}</Button>}
          {detail && detail.status !== 'done' && detail.status !== 'cancelled' && can('surgery:manage') && <Button variant="danger" icon="X" onClick={() => setStatus('cancelled')}>{t('surgeries.cancel')}</Button>}
          <Button variant="ghost" onClick={() => setDetail(null)}>{t('common.close')}</Button>
        </div>
      }>
        {detail && (
          <div className="form-grid">
            <Field label={t('surgeries.patient')}><Input value={pName(detail.patient_id)} disabled /></Field>
            <Field label={t('surgeries.surgeon')}><Input value={detail.surgeon_name || dName(detail.surgeon_id)} disabled /></Field>
            <Field label={t('surgeries.room')}><Input value={rNo(detail.room_id)} disabled /></Field>
            <Field label={t('surgeries.status')}><Badge tone={STATUS_TONE[detail.status]}>{t('surgeries.status.' + detail.status)}</Badge></Field>
            <Field label={t('surgeries.operationName')} className="span-2"><Input value={detail.operation_name} onChange={(e) => setDetail({ ...detail, operation_name: e.target.value })} /></Field>
            <Field label={t('surgeries.operationNameEn')} className="span-2"><Input value={detail.operation_name_en} onChange={(e) => setDetail({ ...detail, operation_name_en: e.target.value })} /></Field>
            <Field label={t('surgeries.anesthesiaType')}><Input value={detail.anesthesia_type} onChange={(e) => setDetail({ ...detail, anesthesia_type: e.target.value })} /></Field>
            <Field label={t('surgeries.durationMin')}><Input type="number" value={detail.duration_min} onChange={(e) => setDetail({ ...detail, duration_min: e.target.value })} /></Field>
            <Field label={t('surgeries.preOpDiagnosis')} className="span-2"><Textarea value={detail.pre_op_diagnosis} onChange={(e) => setDetail({ ...detail, pre_op_diagnosis: e.target.value })} /></Field>
            <Field label={t('surgeries.detail.findings')} className="span-2"><Textarea value={detail.findings} onChange={(e) => setDetail({ ...detail, findings: e.target.value })} /></Field>
            <Field label={t('surgeries.detail.procedureNotes')} className="span-2"><Textarea value={detail.procedure_notes} onChange={(e) => setDetail({ ...detail, procedure_notes: e.target.value })} /></Field>
            <Field label={t('surgeries.detail.complications')} className="span-2"><Textarea value={detail.complications} onChange={(e) => setDetail({ ...detail, complications: e.target.value })} /></Field>
            <Field label={t('surgeries.detail.postOpDiagnosis')} className="span-2"><Textarea value={detail.post_op_diagnosis} onChange={(e) => setDetail({ ...detail, post_op_diagnosis: e.target.value })} /></Field>
            {can('surgery:manage') && <Button icon="Save" onClick={updateReport}>{t('common.save')}</Button>}
          </div>
        )}
      </Modal>
    </FadeIn>
  );
}
