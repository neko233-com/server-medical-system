import { useEffect, useState, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Badge, Select, Textarea } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';

const MODALITIES = ['CT', 'XR', 'MRI', 'US', 'ENDO', 'OTHER'];
const STATUS_TONE = { ordered: 'info', done: 'warning', reported: 'success' };

// AuthImage fetches an authenticated image endpoint as a blob and renders it,
// because <img> cannot attach the Bearer header.
function AuthImage({ id, className }) {
  const [url, setUrl] = useState(null);
  useEffect(() => {
    let revoked = false;
    client.get(`/imaging/images/${id}/file`, { responseType: 'blob' }).then((r) => {
      if (!revoked) setUrl(URL.createObjectURL(r.data));
    }).catch(() => {});
    return () => { revoked = true; if (url) URL.revokeObjectURL(url); };
  }, [id]);
  return url ? <img src={url} className={className} alt="" /> : <div className="img-ph" />;
}

export default function Imaging() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const [items, setItems] = useState([]);
  const [patients, setPatients] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [detail, setDetail] = useState(null);
  const [uploading, setUploading] = useState(false);
  const fileRef = useRef(null);
  const [form, setForm] = useState({ patient_id: 0, modality: 'CT', body_part: '', room_id: 0, study_date: '', indication: '' });

  const load = useCallback(() => {
    setLoading(true);
    client.get('/imaging', { params: { q, page, page_size: 10 } }).then((r) => {
      setItems(r.data.items); setTotal(r.data.pagination.total);
    }).finally(() => setLoading(false));
  }, [q, page]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    client.get('/patients', { params: { page_size: 200 } }).then((r) => setPatients(r.data.items)).catch(() => {});
    client.get('/rooms', { params: { page_size: 100 } }).then((r) => setRooms(r.data.items)).catch(() => {});
  }, []);

  const create = async () => {
    const payload = {
      ...form, patient_id: Number(form.patient_id),
      room_id: Number(form.room_id) || undefined,
      study_date: form.study_date ? new Date(form.study_date).toISOString() : undefined,
    };
    await client.post('/imaging', payload);
    setModal(false);
    setForm({ patient_id: 0, modality: 'CT', body_part: '', room_id: 0, study_date: '', indication: '' });
    load();
  };

  const openDetail = async (s) => {
    const { data } = await client.get(`/imaging/${s.id}`);
    setDetail(data);
  };

  const onUpload = async (e) => {
    const file = e.target.files?.[0];
    if (!file || !detail) return;
    setUploading(true);
    const fd = new FormData();
    fd.append('file', file);
    fd.append('caption', file.name);
    try {
      await client.post(`/imaging/${detail.id}/images`, fd, { headers: { 'Content-Type': 'multipart/form-data' } });
      const { data } = await client.get(`/imaging/${detail.id}`);
      setDetail(data);
    } finally {
      setUploading(false);
      if (fileRef.current) fileRef.current.value = '';
    }
  };

  const delImage = async (imgId) => {
    await client.delete(`/imaging/images/${imgId}`);
    const { data } = await client.get(`/imaging/${detail.id}`);
    setDetail(data);
  };

  const setStatus = async (status) => {
    const { data } = await client.put(`/imaging/${detail.id}/status`, {
      status, findings: detail.findings, impression: detail.impression,
    });
    setDetail(data.item);
    load();
  };

  const saveReport = async () => {
    await client.put(`/imaging/${detail.id}`, { findings: detail.findings, impression: detail.impression });
  };

  const pName = (id) => patients.find((p) => p.id === id)?.name || `#${id}`;
  const rNo = (id) => rooms.find((r) => r.id === id)?.room_no || (id ? `#${id}` : '—');

  const columns = [
    { key: 'study_no', label: t('imaging.table.no'), render: (s) => <strong>{s.study_no}</strong> },
    { key: 'patient', label: t('imaging.table.patient'), render: (s) => pName(s.patient_id) },
    { key: 'modality', label: t('imaging.table.modality'), render: (s) => <Badge tone="info">{s.modality}</Badge> },
    { key: 'body_part', label: t('imaging.table.bodyPart') },
    { key: 'study_date', label: t('imaging.table.studyDate'), render: (s) => (s.study_date || '').slice(0, 10) },
    { key: 'status', label: t('imaging.table.status'), render: (s) => <Badge tone={STATUS_TONE[s.status]}>{t('imaging.status.' + s.status)}</Badge> },
    {
      key: 'actions', label: t('common.actions'), align: 'right',
      render: (s) => <Button size="sm" variant="ghost" icon="Eye" onClick={() => openDetail(s)}>{t('common.detail')}</Button>,
    },
  ];

  return (
    <FadeIn>
      <div className="toolbar">
        <Input className="input--search" placeholder={t('imaging.search')} value={q} onChange={(e) => { setQ(e.target.value); setPage(1); }} />
        {can('imaging:upload') && <Button icon="Plus" onClick={() => setModal(true)}>{t('imaging.new')}</Button>}
      </div>

      <Card padding={false}>
        {loading ? <div className="center-pad"><Spinner /></div> : <Table columns={columns} rows={items} empty={t('common.empty')} />}
      </Card>

      <Modal open={modal} onClose={() => setModal(false)} title={t('imaging.new')} footer={
        <>
          <Button variant="ghost" onClick={() => setModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={create}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('imaging.patient')} className="span-2">
            <Select value={form.patient_id} onChange={(e) => setForm({ ...form, patient_id: Number(e.target.value) })}>
              <option value={0}>—</option>{patients.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
            </Select>
          </Field>
          <Field label={t('imaging.modality')}>
            <Select value={form.modality} onChange={(e) => setForm({ ...form, modality: e.target.value })}>{MODALITIES.map((m) => <option key={m} value={m}>{t('imaging.modality.' + m)}</option>)}</Select>
          </Field>
          <Field label={t('imaging.room')}>
            <Select value={form.room_id} onChange={(e) => setForm({ ...form, room_id: Number(e.target.value) })}>
              <option value={0}>—</option>{rooms.map((r) => <option key={r.id} value={r.id}>{r.room_no} {r.name_zh}</option>)}
            </Select>
          </Field>
          <Field label={t('imaging.bodyPart')} className="span-2"><Input value={form.body_part} onChange={(e) => setForm({ ...form, body_part: e.target.value })} placeholder="胸部 / 头颅 / 腹部…" /></Field>
          <Field label={t('imaging.studyDate')}><Input type="date" value={form.study_date} onChange={(e) => setForm({ ...form, study_date: e.target.value })} /></Field>
          <Field label={t('imaging.indication')} className="span-2"><Input value={form.indication} onChange={(e) => setForm({ ...form, indication: e.target.value })} /></Field>
        </div>
      </Modal>

      <Modal open={!!detail} onClose={() => setDetail(null)} width={760} title={`${t('imaging.title')} · ${detail?.study_no}`} footer={
        <div className="row-actions" style={{ justifyContent: 'flex-end' }}>
          {detail && detail.status === 'ordered' && can('imaging:upload') && <Button icon="Check" onClick={() => setStatus('done')}>{t('imaging.status.done')}</Button>}
          {detail && detail.status !== 'reported' && can('imaging:report') && <Button icon="FileText" onClick={() => setStatus('reported')}>{t('imaging.detail.report')}</Button>}
          <Button variant="ghost" onClick={() => setDetail(null)}>{t('common.close')}</Button>
        </div>
      }>
        {detail && (
          <div className="imaging-detail">
            <div className="form-grid" style={{ marginBottom: 12 }}>
              <Field label={t('imaging.patient')}><Input value={pName(detail.patient_id)} disabled /></Field>
              <Field label={t('imaging.modality')}><Input value={detail.modality} disabled /></Field>
              <Field label={t('imaging.bodyPart')}><Input value={detail.body_part} onChange={(e) => setDetail({ ...detail, body_part: e.target.value })} /></Field>
              <Field label={t('imaging.room')}><Input value={rNo(detail.room_id)} disabled /></Field>
            </div>
            <div className="form-grid" style={{ marginBottom: 12 }}>
              <Field label={t('imaging.detail.findings')} className="span-2"><Textarea value={detail.findings} onChange={(e) => setDetail({ ...detail, findings: e.target.value })} /></Field>
              <Field label={t('imaging.detail.impression')} className="span-2"><Textarea value={detail.impression} onChange={(e) => setDetail({ ...detail, impression: e.target.value })} />{can('imaging:report') && <Button size="sm" icon="Save" onClick={saveReport} style={{ marginTop: 8 }}>{t('common.save')}</Button>}</Field>
              {detail.reported_by && <Field label={t('imaging.detail.reportedBy')}><Input value={detail.reported_by} disabled /></Field>}
            </div>

            <div className="imaging-upload">
              <input ref={fileRef} type="file" accept="image/*,.dcm" onChange={onUpload} hidden />
              <Button icon="Upload" onClick={() => fileRef.current?.click()} disabled={uploading || !can('imaging:upload')}>{uploading ? t('common.loading') : t('imaging.detail.upload')}</Button>
              <small>{t('imaging.uploadHint')}</small>
            </div>

            <div className="film-grid">
              {detail.images?.length === 0 && <p className="muted">{t('common.empty')}</p>}
              {detail.images?.map((img) => (
                <div key={img.id} className="film">
                  <AuthImage id={img.id} className="film__img" />
                  <div className="film__cap">{img.caption}</div>
                  {can('imaging:upload') && <Button size="sm" variant="danger" icon="Trash" onClick={() => delImage(img.id)} />}
                </div>
              ))}
            </div>
          </div>
        )}
      </Modal>
    </FadeIn>
  );
}
