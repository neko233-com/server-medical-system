import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Badge, Select, StatCard } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import { formatMoney } from '../utils/format';
import FadeIn from '../components/anim/FadeIn';

/**
 * Billing — 收费结算
 *
 * An invoice (结算单) is the hospital's financial record of everything a
 * patient is charged for: consultation, medicines, exams, surgery and bed fees.
 *
 * Lifecycle: pending → paid → refunded (or cancelled while still pending).
 * Once an invoice is paid it becomes immutable — it can no longer be edited
 * or deleted, so the ledger stays auditable. Refunds are recorded as a
 * status transition rather than a deletion.
 */

// Invoice status → badge tone.
const INV_TONE = {
  pending: 'amber',
  paid: 'green',
  refunded: 'red',
  cancelled: 'muted',
};

// Charge categories available when adding a line item.
const ITEM_TYPES = ['consult', 'medicine', 'exam', 'surgery', 'bed', 'other'];

// Payment methods recognised by the settlement desk.
const PAY_METHODS = ['cash', 'card', 'wechat', 'alipay', 'insurance'];

export default function Billing() {
  const { t } = useTranslation();
  const { can } = useAuth();
  const canManage = can('billing:manage');

  const [items, setItems] = useState([]);
  const [patients, setPatients] = useState([]);
  const [doctors, setDoctors] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [createModal, setCreateModal] = useState(false);
  const [detail, setDetail] = useState(null);
  const [itemModal, setItemModal] = useState(false);
  const [payModal, setPayModal] = useState(false);
  const [payMethod, setPayMethod] = useState('cash');
  const [form, setForm] = useState({ patient_id: 0, department: '', doctor_id: 0, discount: 0, remark: '' });
  const [itemForm, setItemForm] = useState({ item_type: 'consult', name_zh: '', name_en: '', qty: 1, unit: '次', unit_price: 0 });

  // Loads a page of invoices.
  const load = useCallback(() => {
    setLoading(true);
    client
      .get('/invoices', { params: { page, page_size: 20, status: statusFilter || undefined } })
      .then((r) => {
        setItems(r.data.items || []);
        setTotal(r.data.pagination?.total || 0);
      })
      .catch(() => setError(t('common.load_failed')))
      .finally(() => setLoading(false));
  }, [page, statusFilter, t]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    client.get('/patients', { params: { page_size: 200 } }).then((r) => setPatients(r.data.items || [])).catch(() => {});
    client.get('/rbac/directory').then((r) => setDoctors(r.data || [])).catch(() => {});
  }, []);

  // Opens an invoice with its line items.
  const openDetail = async (inv) => {
    setError('');
    try {
      const { data } = await client.get(`/invoices/${inv.id}`);
      setDetail(data);
    } catch (e) {
      setError(e?.response?.data?.error || t('common.load_failed'));
    }
  };

  // Creates a new (pending) invoice.
  const create = async () => {
    setError('');
    try {
      await client.post('/invoices', {
        patient_id: Number(form.patient_id),
        doctor_id: Number(form.doctor_id) || undefined,
        department: form.department || undefined,
        discount: Number(form.discount) || 0,
        remark: form.remark,
      });
      setCreateModal(false);
      setForm({ patient_id: 0, department: '', doctor_id: 0, discount: 0, remark: '' });
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Adds a chargeable line to an invoice.
  const addItem = async () => {
    setError('');
    try {
      await client.post(`/invoices/${detail.id}/items`, {
        ...itemForm,
        qty: Number(itemForm.qty) || 0,
        unit_price: Number(itemForm.unit_price) || 0,
      });
      setItemModal(false);
      setItemForm({ item_type: 'consult', name_zh: '', name_en: '', qty: 1, unit: '次', unit_price: 0 });
      await openDetail(detail);
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Removes a line and refreshes the totals.
  const removeItem = async (itemId) => {
    setError('');
    try {
      await client.delete(`/invoices/items/${itemId}`);
      await openDetail(detail);
    } catch (e) {
      setError(e?.response?.data?.error || t('common.delete_failed'));
    }
  };

  // Settles the invoice.
  const pay = async () => {
    setError('');
    try {
      await client.post(`/invoices/${detail.id}/pay`, { payment_method: payMethod });
      setPayModal(false);
      await openDetail(detail);
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Refunds a settled invoice.
  const refund = async () => {
    setError('');
    try {
      await client.post(`/invoices/${detail.id}/refund`);
      await openDetail(detail);
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.save_failed'));
    }
  };

  // Deletes an invoice — only allowed while it is still pending.
  const remove = async (id) => {
    setError('');
    try {
      await client.delete(`/invoices/${id}`);
      load();
    } catch (e) {
      setError(e?.response?.data?.error || t('common.delete_failed'));
    }
  };

  const columns = [
    { key: 'invoice_no', label: t('billing.table.no'), render: (i) => <strong>{i.invoice_no}</strong> },
    { key: 'patient_name', label: t('billing.table.patient'), render: (i) => i.patient_name || '—' },
    { key: 'department', label: t('billing.table.department'), render: (i) => i.department || '—' },
    {
      key: 'total',
      label: t('billing.table.total'),
      align: 'right',
      render: (i) => <strong>{formatMoney(i.total, i.currency || 'CNY')}</strong>,
    },
    {
      key: 'status',
      label: t('billing.table.status'),
      render: (i) => <Badge tone={INV_TONE[i.status] || 'neutral'}>{t(`billing.status.${i.status}`)}</Badge>,
    },
    {
      key: 'payment_method',
      label: t('billing.table.method'),
      render: (i) => (i.payment_method ? t(`billing.method.${i.payment_method}`) : '—'),
    },
    {
      key: 'created_at',
      label: t('billing.table.createdAt'),
      render: (i) => new Date(i.created_at).toLocaleString(),
    },
    {
      key: 'ops',
      label: t('common.actions'),
      align: 'right',
      render: (i) => (
        <div className="btn-row">
          <Button size="sm" variant="secondary" onClick={() => openDetail(i)}>{t('common.detail')}</Button>
          {canManage && i.status === 'pending' && (
            <Button size="sm" variant="danger" icon="Trash" onClick={() => remove(i.id)} />
          )}
        </div>
      ),
    },
  ];

  const detailColumns = [
    { key: 'item_type', label: t('billing.detail.type'), render: (it) => t(`billing.itemType.${it.item_type}`) },
    { key: 'name_zh', label: t('billing.detail.name'), render: (it) => it.name_zh || it.name_en || '—' },
    { key: 'qty', label: t('billing.detail.qty'), align: 'right', render: (it) => `${it.qty} ${it.unit || ''}` },
    { key: 'unit_price', label: t('billing.detail.unitPrice'), align: 'right', render: (it) => formatMoney(it.unit_price) },
    { key: 'amount', label: t('billing.detail.amount'), align: 'right', render: (it) => <strong>{formatMoney(it.amount)}</strong> },
    {
      key: 'ops',
      label: t('common.actions'),
      align: 'right',
      render: (it) => (
        canManage && detail?.status === 'pending'
          ? <Button size="sm" variant="danger" icon="Trash" onClick={() => removeItem(it.id)} />
          : null
      ),
    },
  ];

  return (
    <FadeIn>
      {error && <div className="alert alert--error">{error}</div>}

      <div className="toolbar">
        <Select value={statusFilter} onChange={(e) => { setPage(1); setStatusFilter(e.target.value); }}>
          <option value="">{t('billing.filter.all')}</option>
          {['pending', 'paid', 'refunded', 'cancelled'].map((s) => (
            <option key={s} value={s}>{t(`billing.status.${s}`)}</option>
          ))}
        </Select>
        <span className="muted">{t('common.total')}: {total}</span>
        {canManage && (
          <Button icon="Receipt" onClick={() => setCreateModal(true)}>{t('billing.new')}</Button>
        )}
      </div>

      <Card padding={false}>
        {loading ? (
          <div className="center-pad"><Spinner size={26} /></div>
        ) : (
          <Table columns={columns} rows={items} empty={t('billing.empty')} />
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

      {/* ---- 新建结算单 ---- */}
      <Modal open={createModal} onClose={() => setCreateModal(false)} title={t('billing.new')} footer={
        <>
          <Button variant="ghost" onClick={() => setCreateModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={create}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('appointments.patient')}>
            <Select value={form.patient_id} onChange={(e) => setForm({ ...form, patient_id: e.target.value })}>
              <option value={0}>{t('appointments.patientPlaceholder')}</option>
              {patients.map((p) => <option key={p.id} value={p.id}>{p.name} ({p.code})</option>)}
            </Select>
          </Field>
          <Field label={t('appointments.doctor')}>
            <Select value={form.doctor_id} onChange={(e) => setForm({ ...form, doctor_id: e.target.value })}>
              <option value={0}>{t('common.none')}</option>
              {doctors.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
            </Select>
          </Field>
          <Field label={t('appointments.department')}>
            <Input value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} />
          </Field>
          <Field label={t('billing.discount')}>
            <Input type="number" min="0" step="0.01" value={form.discount}
              onChange={(e) => setForm({ ...form, discount: e.target.value })} />
          </Field>
          <Field label={t('billing.remark')} className="span-2">
            <Input value={form.remark} onChange={(e) => setForm({ ...form, remark: e.target.value })} />
          </Field>
        </div>
      </Modal>

      {/* ---- 结算单明细 ---- */}
      <Modal
        open={!!detail}
        onClose={() => setDetail(null)}
        title={detail ? `${t('billing.detail.title')} · ${detail.invoice_no}` : ''}
        width={720}
        footer={detail && (
          <>
            <Button variant="ghost" onClick={() => setDetail(null)}>{t('common.close')}</Button>
            {canManage && detail.status === 'pending' && (
              <>
                <Button variant="secondary" icon="Plus" onClick={() => setItemModal(true)}>{t('billing.addItem')}</Button>
                <Button icon="Check" onClick={() => setPayModal(true)}>{t('billing.pay')}</Button>
              </>
            )}
            {canManage && detail.status === 'paid' && (
              <Button variant="danger" icon="Undo" onClick={refund}>{t('billing.refund')}</Button>
            )}
          </>
        )}
      >
        {detail && (
          <>
            <div className="stat-grid" style={{ marginBottom: 16 }}>
              <StatCard label={t('billing.detail.subtotal')} value={formatMoney(detail.subtotal, detail.currency)} icon="Receipt" tone="primary" />
              <StatCard label={t('billing.detail.discount')} value={formatMoney(detail.discount, detail.currency)} icon="Minus" tone="amber" />
              <StatCard label={t('billing.detail.total')} value={formatMoney(detail.total, detail.currency)} icon="Wallet" tone="green" />
              <StatCard label={t('billing.detail.status')} value={t(`billing.status.${detail.status}`)} icon="ClipboardList" tone="violet" />
            </div>
            <Table columns={detailColumns} rows={detail.items || []} empty={t('billing.detail.noItems')} />
          </>
        )}
      </Modal>

      {/* ---- 添加收费明细 ---- */}
      <Modal open={itemModal} onClose={() => setItemModal(false)} title={t('billing.addItem')} footer={
        <>
          <Button variant="ghost" onClick={() => setItemModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={addItem}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('billing.detail.type')}>
            <Select value={itemForm.item_type} onChange={(e) => setItemForm({ ...itemForm, item_type: e.target.value })}>
              {ITEM_TYPES.map((tp) => <option key={tp} value={tp}>{t(`billing.itemType.${tp}`)}</option>)}
            </Select>
          </Field>
          <Field label={t('billing.detail.name')}>
            <Input value={itemForm.name_zh} onChange={(e) => setItemForm({ ...itemForm, name_zh: e.target.value })} />
          </Field>
          <Field label={t('billing.detail.nameEn')}>
            <Input value={itemForm.name_en} onChange={(e) => setItemForm({ ...itemForm, name_en: e.target.value })} />
          </Field>
          <Field label={t('billing.detail.unit')}>
            <Input value={itemForm.unit} onChange={(e) => setItemForm({ ...itemForm, unit: e.target.value })} />
          </Field>
          <Field label={t('billing.detail.qty')}>
            <Input type="number" min="0" step="0.01" value={itemForm.qty}
              onChange={(e) => setItemForm({ ...itemForm, qty: e.target.value })} />
          </Field>
          <Field label={t('billing.detail.unitPrice')}>
            <Input type="number" min="0" step="0.01" value={itemForm.unit_price}
              onChange={(e) => setItemForm({ ...itemForm, unit_price: e.target.value })} />
          </Field>
        </div>
      </Modal>

      {/* ---- 收款 ---- */}
      <Modal open={payModal} onClose={() => setPayModal(false)} title={t('billing.pay')} footer={
        <>
          <Button variant="ghost" onClick={() => setPayModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Check" onClick={pay}>{t('billing.confirmPay')}</Button>
        </>
      }>
        {detail && (
          <p className="muted" style={{ marginBottom: 12 }}>
            {t('billing.payAmount')}: <strong>{formatMoney(detail.total, detail.currency)}</strong>
          </p>
        )}
        <Field label={t('billing.table.method')}>
          <Select value={payMethod} onChange={(e) => setPayMethod(e.target.value)}>
            {PAY_METHODS.map((m) => <option key={m} value={m}>{t(`billing.method.${m}`)}</option>)}
          </Select>
        </Field>
      </Modal>
    </FadeIn>
  );
}
