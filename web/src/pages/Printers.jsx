import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Badge, Select } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';

const TYPES = ['thermal', 'receipt', 'a4', 'label'];
const CONNS = ['network', 'usb', 'browser'];

export default function Printers() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const [items, setItems] = useState([]);
  const [doctors, setDoctors] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [resolved, setResolved] = useState(null);
  const [form, setForm] = useState({ name: '', type: 'thermal', connection: 'network', ip_address: '', port: '', doctor_id: 0, room_id: 0, is_default: false, status: 'active', notes: '' });

  const load = useCallback(() => {
    setLoading(true);
    client.get('/printers', { params: { q, page, page_size: 10 } }).then((r) => {
      setItems(r.data.items); setTotal(r.data.pagination.total);
    }).finally(() => setLoading(false));
    client.get('/print-jobs', { params: { page_size: 8 } }).then((r) => setJobs(r.data.items)).catch(() => {});
  }, [q, page]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    client.get('/rbac/directory').then((r) => setDoctors(r.data)).catch(() => {});
    client.get('/rooms', { params: { page_size: 100 } }).then((r) => setRooms(r.data.items)).catch(() => {});
  }, []);

  const save = async () => {
    const payload = { ...form, doctor_id: Number(form.doctor_id) || 0, room_id: Number(form.room_id) || 0 };
    if (form.id) await client.put(`/printers/${form.id}`, payload);
    else await client.post('/printers', payload);
    setModal(false);
    setForm({ name: '', type: 'thermal', connection: 'network', ip_address: '', port: '', doctor_id: 0, room_id: 0, is_default: false, status: 'active', notes: '' });
    load();
  };

  const open = (p) => setForm({ id: p.id, name: p.name, type: p.type, connection: p.connection, ip_address: p.ip_address, port: p.port, doctor_id: p.doctor_id, room_id: p.room_id, is_default: p.is_default, status: p.status, notes: p.notes });

  const remove = async (id) => { await client.delete(`/printers/${id}`); load(); };
  const setDefault = async (id) => { await client.put(`/printers/${id}/default`); load(); };

  const resolve = () => {
    client.get('/printers/resolve').then((r) => setResolved(r.data)).catch(() => setResolved(null));
  };

  const doctorName = (id) => doctors.find((d) => d.id === id)?.name || (id ? `#${id}` : '—');
  const roomNo = (id) => rooms.find((r) => r.id === id)?.room_no || (id ? `#${id}` : '—');

  const columns = [
    { key: 'name', label: t('printers.table.name'), render: (p) => <strong>{p.name}{p.is_default && <Badge tone="warning" style={{ marginLeft: 6 }}>{t('printers.table.default')}</Badge>}</strong> },
    { key: 'type', label: t('printers.table.type'), render: (p) => <Badge tone="info">{t('printers.type.' + p.type)}</Badge> },
    { key: 'connection', label: t('printers.table.connection'), render: (p) => p.connection },
    { key: 'ip', label: t('printers.table.ip'), render: (p) => p.ip_address + (p.port ? ':' + p.port : '') },
    { key: 'doctor', label: t('printers.table.doctor'), render: (p) => doctorName(p.doctor_id) },
    { key: 'room', label: t('printers.table.room'), render: (p) => roomNo(p.room_id) },
    { key: 'status', label: t('printers.table.status'), render: (p) => <Badge tone={p.status === 'active' ? 'success' : 'muted'}>{t('status.' + (p.status === 'active' ? 'active' : 'inactive'))}</Badge> },
    {
      key: 'actions', label: t('common.actions'), align: 'right',
      render: (p) => (
        <div className="row-actions">
          {!p.is_default && can('printer:manage') && <Button size="sm" variant="ghost" icon="Check" onClick={() => setDefault(p.id)} />}
          {can('printer:manage') && <Button size="sm" variant="ghost" icon="Edit" onClick={() => open(p)} />}
          {can('printer:manage') && <Button size="sm" variant="danger" icon="Trash" onClick={() => remove(p.id)} />}
        </div>
      ),
    },
  ];

  return (
    <FadeIn>
      <div className="toolbar">
        <Input className="input--search" placeholder={t('printers.search')} value={q} onChange={(e) => { setQ(e.target.value); setPage(1); }} />
        <Button icon="Target" onClick={resolve}>{t('printers.resolve')}</Button>
        {can('printer:manage') && <Button icon="Plus" onClick={() => { setForm({ name: '', type: 'thermal', connection: 'network', ip_address: '', port: '', doctor_id: 0, room_id: 0, is_default: false, status: 'active', notes: '' }); setModal(true); }}>{t('printers.new')}</Button>}
      </div>

      {resolved && (
        <Card>
          <p style={{ margin: 0 }}><strong>{t('printers.resolve')}:</strong> {resolved.name}（{t('printers.room')}: {resolved.room_no || '—'} · {t('printers.doctor')}: {resolved.doctor_name || '—'}）</p>
        </Card>
      )}

      <div className="grid-2">
        <Card padding={false}>
          {loading ? <div className="center-pad"><Spinner /></div> : <Table columns={columns} rows={items} empty={t('common.empty')} />}
        </Card>
        <Card title={t('printers.jobs')}>
          {jobs.length === 0 ? <Empty text={t('common.empty')} /> : (
            <ul className="job-list">
              {jobs.map((j) => (
                <li key={j.id}>
                  <div><strong>{j.printer_name}</strong> <Badge tone="muted">{j.doc_type}</Badge></div>
                  <small>{doctorName(j.doctor_id)} · {roomNo(j.room_id)} · {j.summary}</small>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>

      <Modal open={modal} onClose={() => setModal(false)} title={t('printers.new')} footer={
        <>
          <Button variant="ghost" onClick={() => setModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={save}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('printers.name')} className="span-2"><Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field>
          <Field label={t('printers.type')}>
            <Select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}>{TYPES.map((x) => <option key={x} value={x}>{t('printers.type.' + x)}</option>)}</Select>
          </Field>
          <Field label={t('printers.connection')}>
            <Select value={form.connection} onChange={(e) => setForm({ ...form, connection: e.target.value })}>{CONNS.map((x) => <option key={x} value={x}>{x}</option>)}</Select>
          </Field>
          <Field label={t('printers.ip')}><Input value={form.ip_address} onChange={(e) => setForm({ ...form, ip_address: e.target.value })} placeholder="192.168.1.10" /></Field>
          <Field label={t('printers.port')}><Input value={form.port} onChange={(e) => setForm({ ...form, port: e.target.value })} placeholder="9100" /></Field>
          <Field label={t('printers.doctor')}>
            <Select value={form.doctor_id} onChange={(e) => setForm({ ...form, doctor_id: Number(e.target.value) })}><option value={0}>—</option>{doctors.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}</Select>
          </Field>
          <Field label={t('printers.room')}>
            <Select value={form.room_id} onChange={(e) => setForm({ ...form, room_id: Number(e.target.value) })}><option value={0}>—</option>{rooms.map((r) => <option key={r.id} value={r.id}>{r.room_no} {r.name_zh}</option>)}</Select>
          </Field>
          <Field label={t('printers.status')}>
            <Select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}><option value="active">{t('status.active')}</option><option value="offline">{t('status.inactive')}</option></Select>
          </Field>
          <Field label={t('printers.isDefault')}>
            <Select value={form.is_default ? '1' : '0'} onChange={(e) => setForm({ ...form, is_default: e.target.value === '1' })}><option value="0">{t('common.no')}</option><option value="1">{t('common.yes')}</option></Select>
          </Field>
          <Field label={t('printers.notes')} className="span-2"><Input value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
        </div>
      </Modal>
    </FadeIn>
  );
}
