import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Empty, Select, Badge } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';

const TYPES = ['consultation', 'imaging', 'surgery', 'emergency', 'ward'];

export default function Rooms() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const [items, setItems] = useState([]);
  const [doctors, setDoctors] = useState([]);
  const [printers, setPrinters] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [form, setForm] = useState({ room_no: '', name_zh: '', type: 'consultation', department: '', floor: '', doctor_id: 0, printer_id: 0, status: 'active', description: '' });

  const load = useCallback(() => {
    setLoading(true);
    client.get('/rooms', { params: { q, page, page_size: 10 } }).then((r) => {
      setItems(r.data.items);
      setTotal(r.data.pagination.total);
    }).finally(() => setLoading(false));
  }, [q, page]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    client.get('/rbac/directory').then((r) => setDoctors(r.data)).catch(() => {});
    if (can('printer:read')) client.get('/printers', { params: { page_size: 100 } }).then((r) => setPrinters(r.data.items)).catch(() => {});
  }, [can]);

  const save = async () => {
    const payload = { ...form, doctor_id: Number(form.doctor_id) || 0, printer_id: Number(form.printer_id) || 0 };
    if (form.id) {
      await client.put(`/rooms/${form.id}`, payload);
    } else {
      await client.post('/rooms', payload);
    }
    setModal(false);
    setForm({ room_no: '', name_zh: '', type: 'consultation', department: '', floor: '', doctor_id: 0, printer_id: 0, status: 'active', description: '' });
    load();
  };

  const open = (room) => {
    setForm({
      id: room.id, room_no: room.room_no, name_zh: room.name_zh, type: room.type,
      department: room.department, floor: room.floor, doctor_id: room.doctor_id,
      printer_id: room.printer_id, status: room.status, description: room.description,
    });
    setModal(true);
  };

  const remove = async (id) => { await client.delete(`/rooms/${id}`); load(); };

  const assignDoctor = async (room) => {
    const id = window.prompt(t('rooms.assignDoctor') + ' (id)', room.doctor_id || '');
    if (id === null) return;
    await client.put(`/rooms/${room.id}/doctor`, { doctor_id: Number(id) || 0 });
    load();
  };

  const doctorName = (id) => (doctors.find((d) => d.id === id)?.name) || (id ? `#${id}` : '—');
  const printerName = (id) => (printers.find((p) => p.id === id)?.name) || (id ? `#${id}` : '—');

  const columns = [
    { key: 'room_no', label: t('rooms.table.roomNo'), render: (r) => <strong>{r.room_no}</strong> },
    { key: 'name_zh', label: t('rooms.table.name') },
    { key: 'type', label: t('rooms.table.type'), render: (r) => <Badge tone="info">{t('rooms.type.' + r.type)}</Badge> },
    { key: 'department', label: t('rooms.table.department') },
    { key: 'doctor', label: t('rooms.table.doctor'), render: (r) => doctorName(r.doctor_id) },
    { key: 'printer', label: t('rooms.table.printer'), render: (r) => printerName(r.printer_id) },
    { key: 'status', label: t('rooms.table.status'), render: (r) => <Badge tone={r.status === 'active' ? 'success' : 'muted'}>{t('status.' + (r.status === 'active' ? 'active' : 'inactive'))}</Badge> },
    {
      key: 'actions', label: t('common.actions'), align: 'right',
      render: (r) => (
        <div className="row-actions">
          <Button size="sm" variant="ghost" icon="UserCheck" onClick={() => assignDoctor(r)} disabled={!can('room:manage')}>{t('rooms.assignDoctor')}</Button>
          {can('room:manage') && <Button size="sm" variant="ghost" icon="Edit" onClick={() => open(r)} />}
          {can('room:manage') && <Button size="sm" variant="danger" icon="Trash" onClick={() => remove(r.id)} />}
        </div>
      ),
    },
  ];

  return (
    <FadeIn>
      <div className="toolbar">
        <Input className="input--search" placeholder={t('rooms.search')} value={q} onChange={(e) => { setQ(e.target.value); setPage(1); }} />
        {can('room:manage') && <Button icon="Plus" onClick={() => { setForm({ room_no: '', name_zh: '', type: 'consultation', department: '', floor: '', doctor_id: 0, printer_id: 0, status: 'active', description: '' }); setModal(true); }}>{t('rooms.new')}</Button>}
      </div>

      <Card padding={false}>
        {loading ? <div className="center-pad"><Spinner /></div> : <Table columns={columns} rows={items} empty={t('common.empty')} />}
      </Card>

      <Modal open={modal} onClose={() => setModal(false)} title={form.id ? t('rooms.new') : t('rooms.new')} footer={
        <>
          <Button variant="ghost" onClick={() => setModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={save}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('rooms.roomNo')}><Input value={form.room_no} onChange={(e) => setForm({ ...form, room_no: e.target.value })} /></Field>
          <Field label={t('rooms.name')}><Input value={form.name_zh} onChange={(e) => setForm({ ...form, name_zh: e.target.value })} /></Field>
          <Field label={t('rooms.type')}>
            <Select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}>
              {TYPES.map((tp) => <option key={tp} value={tp}>{t('rooms.type.' + tp)}</option>)}
            </Select>
          </Field>
          <Field label={t('rooms.department')}><Input value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} /></Field>
          <Field label={t('rooms.floor')}><Input value={form.floor} onChange={(e) => setForm({ ...form, floor: e.target.value })} /></Field>
          <Field label={t('rooms.doctor')}>
            <Select value={form.doctor_id} onChange={(e) => setForm({ ...form, doctor_id: Number(e.target.value) })}>
              <option value={0}>—</option>
              {doctors.map((d) => <option key={d.id} value={d.id}>{d.name}（{d.role}）</option>)}
            </Select>
          </Field>
          <Field label={t('rooms.printer')}>
            <Select value={form.printer_id} onChange={(e) => setForm({ ...form, printer_id: Number(e.target.value) })}>
              <option value={0}>—</option>
              {printers.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </Select>
          </Field>
          <Field label={t('rooms.status')}>
            <Select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
              <option value="active">{t('status.active')}</option>
              <option value="maintenance">{t('status.inactive')}</option>
            </Select>
          </Field>
          <Field label={t('rooms.description')} className="span-2"><Input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field>
        </div>
      </Modal>
    </FadeIn>
  );
}
