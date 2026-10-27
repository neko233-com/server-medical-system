import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Empty, Badge, Toggle } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import Reicon from '../components/Reicon';
import FadeIn from '../components/anim/FadeIn';
import { medicineName, categoryName, formatMoney } from '../utils/format';

const emptyForm = {
  code: '', name_zh: '', name_en: '', category: '', category_en: '', spec: '',
  manufacturer: '', unit: '', price: 0, currency: 'CNY', is_active: true,
};

export default function Medicines() {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';

  const [items, setItems] = useState([]);
  const [q, setQ] = useState('');
  const [loading, setLoading] = useState(true);
  const [modal, setModal] = useState(false);
  const [priceModal, setPriceModal] = useState(null);
  const [form, setForm] = useState(emptyForm);

  const load = useCallback(() => {
    setLoading(true);
    client
      .get('/medicines', { params: { q, page: 1, page_size: 20 } })
      .then((r) => setItems(r.data.items || []))
      .finally(() => setLoading(false));
  }, [q]);

  useEffect(() => {
    load();
  }, [load]);

  const openNew = () => {
    setForm(emptyForm);
    setModal(true);
  };
  const openEdit = (m) => {
    setForm({ ...m, is_active: m.is_active ?? true });
    setModal(true);
  };
  const save = async () => {
    const payload = { ...form, price: Number(form.price) };
    if (form.id) await client.put(`/medicines/${form.id}`, payload);
    else await client.post('/medicines', payload);
    setModal(false);
    load();
  };
  const remove = async (m) => {
    if (!window.confirm(t('medicines.confirmDelete'))) return;
    await client.delete(`/medicines/${m.id}`);
    load();
  };
  const savePrice = async () => {
    await client.put(`/medicines/${priceModal.id}/price`, {
      price: Number(priceModal.price),
      currency: priceModal.currency,
    });
    setPriceModal(null);
    load();
  };

  const columns = [
    { key: 'code', label: t('medicines.code') },
    { key: 'name', label: t('medicines.name'), render: (r) => <strong>{medicineName(r, lang)}</strong> },
    { key: 'category', label: t('medicines.category'), render: (r) => <Badge tone="neutral">{categoryName(r, lang)}</Badge> },
    { key: 'spec', label: t('medicines.spec') },
    { key: 'manufacturer', label: t('medicines.manufacturer') },
    { key: 'price', label: t('medicines.price'), align: 'right', render: (r) => formatMoney(r.price, r.currency) },
    {
      key: 'active',
      label: t('medicines.active'),
      align: 'center',
      render: (r) =>
        r.is_active ? <Badge tone="green">{t('status.active')}</Badge> : <Badge tone="red">{t('status.inactive')}</Badge>,
    },
  ];

  if (isAdmin) {
    columns.push({
      key: 'actions',
      label: t('common.actions'),
      align: 'right',
      render: (r) => (
        <div className="row-actions">
          <Button size="sm" variant="ghost" icon="Edit" onClick={() => openEdit(r)} />
          <Button size="sm" variant="ghost" icon="Dollar" onClick={() => setPriceModal({ id: r.id, price: r.price, currency: r.currency })} />
          <Button size="sm" variant="ghost" icon="Trash" onClick={() => remove(r)} />
        </div>
      ),
    });
  }

  return (
    <FadeIn>
      <div className="toolbar">
        <Input
          className="input--search"
          placeholder={t('medicines.search')}
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        {isAdmin && (
          <Button icon="Plus" onClick={openNew}>
            {t('medicines.new')}
          </Button>
        )}
      </div>
      {!isAdmin && (
        <div className="alert alert--info">
          <Reicon name="AlertTriangle" size={15} /> {t('medicines.adminOnly')}
        </div>
      )}

      <Card padding={false}>
        {loading ? (
          <div className="center-pad">
            <Spinner />
          </div>
        ) : (
          <Table columns={columns} rows={items} empty={t('medicines.empty')} />
        )}
      </Card>

      <Modal
        open={modal}
        onClose={() => setModal(false)}
        title={form.id ? t('medicines.editTitle') : t('medicines.addTitle')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setModal(false)}>
              {t('common.cancel')}
            </Button>
            <Button icon="Save" onClick={save}>
              {t('common.save')}
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label={`${t('medicines.code')}`}>
            <Input value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} />
          </Field>
          <Field label={`${t('medicines.name')} (中文)`}>
            <Input value={form.name_zh} onChange={(e) => setForm({ ...form, name_zh: e.target.value })} />
          </Field>
          <Field label={`${t('medicines.name')} (EN)`}>
            <Input value={form.name_en} onChange={(e) => setForm({ ...form, name_en: e.target.value })} />
          </Field>
          <Field label={`${t('medicines.category')} (中文)`}>
            <Input value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })} />
          </Field>
          <Field label={`${t('medicines.category')} (EN)`}>
            <Input value={form.category_en} onChange={(e) => setForm({ ...form, category_en: e.target.value })} />
          </Field>
          <Field label={t('medicines.spec')}>
            <Input value={form.spec} onChange={(e) => setForm({ ...form, spec: e.target.value })} />
          </Field>
          <Field label={t('medicines.manufacturer')}>
            <Input value={form.manufacturer} onChange={(e) => setForm({ ...form, manufacturer: e.target.value })} />
          </Field>
          <Field label={t('medicines.unit')}>
            <Input value={form.unit} onChange={(e) => setForm({ ...form, unit: e.target.value })} />
          </Field>
          <Field label={t('medicines.price')}>
            <Input type="number" value={form.price} onChange={(e) => setForm({ ...form, price: e.target.value })} />
          </Field>
          <Field label={t('settings.currency')}>
            <Input value={form.currency} onChange={(e) => setForm({ ...form, currency: e.target.value })} />
          </Field>
          <Field label={t('medicines.active')} className="span-2">
            <Toggle
              checked={!!form.is_active}
              onChange={(v) => setForm({ ...form, is_active: v })}
              label={form.is_active ? t('status.active') : t('status.inactive')}
            />
          </Field>
        </div>
      </Modal>

      <Modal
        open={!!priceModal}
        onClose={() => setPriceModal(null)}
        title={t('medicines.editPrice')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setPriceModal(null)}>
              {t('common.cancel')}
            </Button>
            <Button icon="Save" onClick={savePrice}>
              {t('common.save')}
            </Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label={t('medicines.price')}>
            <Input
              type="number"
              value={priceModal?.price ?? 0}
              onChange={(e) => setPriceModal({ ...priceModal, price: e.target.value })}
            />
          </Field>
          <Field label={t('settings.currency')}>
            <Input
              value={priceModal?.currency ?? 'CNY'}
              onChange={(e) => setPriceModal({ ...priceModal, currency: e.target.value })}
            />
          </Field>
        </div>
      </Modal>
    </FadeIn>
  );
}
