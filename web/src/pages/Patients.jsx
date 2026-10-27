import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Empty, Select } from '../components/ui';
import { genderLabel } from '../utils/format';
import FadeIn from '../components/anim/FadeIn';

export default function Patients() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [form, setForm] = useState({ name: '', gender: 'M', birth_date: '', phone: '', id_card: '', address: '' });

  const load = useCallback(() => {
    setLoading(true);
    client
      .get('/patients', { params: { q, page, page_size: 10 } })
      .then((r) => {
        setItems(r.data.items);
        setTotal(r.data.pagination.total);
      })
      .finally(() => setLoading(false));
  }, [q, page]);

  useEffect(() => {
    load();
  }, [load]);

  const create = async () => {
    if (!form.name) return;
    await client.post('/patients', form);
    setModal(false);
    setForm({ name: '', gender: 'M', birth_date: '', phone: '', id_card: '', address: '' });
    load();
  };

  const columns = [
    { key: 'name', label: t('patients.table.name'), render: (r) => <strong>{r.name}</strong> },
    { key: 'gender', label: t('patients.table.gender'), render: (r) => genderLabel(r.gender, t) },
    { key: 'phone', label: t('patients.table.phone') },
    { key: 'birth_date', label: t('patients.table.birth') },
    { key: 'created_at', label: t('patients.table.created'), render: (r) => (r.created_at || '').slice(0, 10) },
    {
      key: 'actions',
      label: t('common.actions'),
      align: 'right',
      render: (r) => (
        <Button size="sm" variant="ghost" icon="Stethoscope" onClick={() => navigate(`/consultation?patient=${r.id}`)}>
          {t('patients.view')}
        </Button>
      ),
    },
  ];

  return (
    <FadeIn>
      <div className="toolbar">
        <Input
          className="input--search"
          placeholder={t('patients.searchPlaceholder')}
          value={q}
          onChange={(e) => {
            setQ(e.target.value);
            setPage(1);
          }}
        />
        <Button icon="Plus" onClick={() => setModal(true)}>
          {t('patients.new')}
        </Button>
      </div>

      <Card padding={false}>
        {loading ? (
          <div className="center-pad">
            <Spinner />
          </div>
        ) : (
          <Table columns={columns} rows={items} empty={t('patients.empty')} />
        )}
      </Card>

      <Modal
        open={modal}
        onClose={() => setModal(false)}
        title={t('patients.new')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setModal(false)}>
              {t('common.cancel')}
            </Button>
            <Button icon="Save" onClick={create}>
              {t('patients.form.create')}
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label={t('patients.form.name')}>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <Field label={t('patients.form.gender')}>
            <Select value={form.gender} onChange={(e) => setForm({ ...form, gender: e.target.value })}>
              <option value="M">{t('gender.male')}</option>
              <option value="F">{t('gender.female')}</option>
            </Select>
          </Field>
          <Field label={t('patients.form.birth')}>
            <Input type="date" value={form.birth_date} onChange={(e) => setForm({ ...form, birth_date: e.target.value })} />
          </Field>
          <Field label={t('patients.form.phone')}>
            <Input value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          </Field>
          <Field label={t('patients.form.idcard')}>
            <Input value={form.id_card} onChange={(e) => setForm({ ...form, id_card: e.target.value })} />
          </Field>
          <Field label={t('patients.form.address')} className="span-2">
            <Input value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} />
          </Field>
        </div>
      </Modal>
    </FadeIn>
  );
}
