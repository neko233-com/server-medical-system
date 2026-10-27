import { useEffect, useState, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Textarea, Select, Modal, Spinner, Empty, Badge } from '../components/ui';
import Reicon from '../components/Reicon';
import FadeIn from '../components/anim/FadeIn';
import { medicineName, categoryName, formatMoney } from '../utils/format';

export default function Consultation() {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const [params] = useSearchParams();

  const [patients, setPatients] = useState([]);
  const [patientId, setPatientId] = useState('');
  const [caseForm, setCaseForm] = useState({
    chief_complaint: '',
    present_illness: '',
    past_history: '',
    diagnosis: '',
    notes: '',
  });
  const [rxItems, setRxItems] = useState([]);
  const [medQuery, setMedQuery] = useState('');
  const [medResults, setMedResults] = useState([]);
  const [searching, setSearching] = useState(false);
  const [saving, setSaving] = useState(false);
  const [newOpen, setNewOpen] = useState(false);
  const [newForm, setNewForm] = useState({ name: '', gender: 'M', birth_date: '', phone: '', id_card: '', address: '' });
  const [toast, setToast] = useState('');
  const debounce = useRef(null);

  // Load patient dropdown + preselect from ?patient=ID.
  useEffect(() => {
    client.get('/patients', { params: { page_size: 100 } }).then((r) => setPatients(r.data.items || []));
  }, []);

  useEffect(() => {
    const pid = params.get('patient');
    if (pid) setPatientId(pid);
  }, [params]);

  // Debounced medicine search.
  useEffect(() => {
    if (!medQuery.trim()) {
      setMedResults([]);
      return;
    }
    setSearching(true);
    clearTimeout(debounce.current);
    debounce.current = setTimeout(() => {
      client
        .get('/medicines', { params: { q: medQuery, page_size: 12 } })
        .then((r) => setMedResults(r.data.items || []))
        .finally(() => setSearching(false));
    }, 250);
    return () => clearTimeout(debounce.current);
  }, [medQuery]);

  const addMed = (med) => {
    if (rxItems.find((i) => i.medicine_id === med.id)) return;
    setRxItems((prev) => [
      ...prev,
      { medicine_id: med.id, name: medicineName(med, lang), spec: med.spec, price: med.price, currency: med.currency, quantity: 1, dosage: '', frequency: '' },
    ]);
    setMedQuery('');
    setMedResults([]);
  };

  const updateItem = (idx, patch) => {
    setRxItems((prev) => prev.map((it, i) => (i === idx ? { ...it, ...patch } : it)));
  };

  const total = rxItems.reduce((s, it) => s + (it.price || 0) * (it.quantity || 0), 0);
  const currency = rxItems[0]?.currency || 'CNY';

  const saveCase = async () => {
    if (!patientId) return;
    setSaving(true);
    try {
      await client.post('/cases', { patient_id: Number(patientId), ...caseForm });
      setToast(t('consultation.caseSaved'));
    } finally {
      setSaving(false);
      setTimeout(() => setToast(''), 2500);
    }
  };

  const issueRx = async () => {
    if (!patientId) return;
    if (rxItems.length === 0) return;
    setSaving(true);
    try {
      const { data } = await client.post('/prescriptions', {
        patient_id: Number(patientId),
        items: rxItems.map((i) => ({
          medicine_id: i.medicine_id,
          quantity: i.quantity,
          dosage: i.dosage,
          frequency: i.frequency,
        })),
      });
      setToast(t('consultation.rxCreated') + ' · ' + data.item.prescription_no);
      setRxItems([]);
    } finally {
      setSaving(false);
      setTimeout(() => setToast(''), 3000);
    }
  };

  const createPatient = async () => {
    if (!newForm.name) return;
    const { data } = await client.post('/patients', newForm);
    const id = data.item.id;
    setPatients((prev) => [data.item, ...prev]);
    setPatientId(String(id));
    setNewOpen(false);
    setNewForm({ name: '', gender: 'M', birth_date: '', phone: '', id_card: '', address: '' });
  };

  return (
    <FadeIn className="stack-lg">
      {toast && <div className="toast"><Reicon name="CircleCheck" size={16} /> {toast}</div>}

      <Card title={t('consultation.selectPatient')}>
        <div className="consult-patient">
          <Select value={patientId} onChange={(e) => setPatientId(e.target.value)} className="consult-patient__select">
            <option value="">{t('common.select')}</option>
            {patients.map((p) => (
              <option key={p.id} value={p.id}>{p.name} · {p.phone || '—'}</option>
            ))}
          </Select>
          <Button variant="ghost" icon="Plus" onClick={() => setNewOpen(true)}>{t('consultation.newPatient')}</Button>
        </div>
        {!patientId && <p className="hint-warn"><Reicon name="AlertTriangle" size={15} /> {t('consultation.selectFirst')}</p>}
      </Card>

      <div className="consult-grid">
        <Card title={t('consultation.title')}>
          <div className="form-grid">
            <Field label={t('consultation.chiefComplaint')} className="span-2">
              <Input value={caseForm.chief_complaint} onChange={(e) => setCaseForm({ ...caseForm, chief_complaint: e.target.value })} />
            </Field>
            <Field label={t('consultation.presentIllness')} className="span-2">
              <Textarea rows={2} value={caseForm.present_illness} onChange={(e) => setCaseForm({ ...caseForm, present_illness: e.target.value })} />
            </Field>
            <Field label={t('consultation.pastHistory')} className="span-2">
              <Textarea rows={2} value={caseForm.past_history} onChange={(e) => setCaseForm({ ...caseForm, past_history: e.target.value })} />
            </Field>
            <Field label={t('consultation.diagnosis')}>
              <Input value={caseForm.diagnosis} onChange={(e) => setCaseForm({ ...caseForm, diagnosis: e.target.value })} />
            </Field>
            <Field label={t('consultation.notes')} optional={t('common.optional')}>
              <Input value={caseForm.notes} onChange={(e) => setCaseForm({ ...caseForm, notes: e.target.value })} />
            </Field>
          </div>
          <div className="consult-actions">
            <Button icon="Save" disabled={!patientId || saving} onClick={saveCase}>{t('consultation.submitCase')}</Button>
          </div>
        </Card>

        <Card title={t('consultation.meds')}>
          <div className="med-search">
            <Input
              className="input--search"
              placeholder={t('consultation.medSearch')}
              value={medQuery}
              onChange={(e) => setMedQuery(e.target.value)}
            />
            {searching && <Spinner size={14} />}
          </div>
          {medResults.length > 0 && (
            <div className="med-results">
              {medResults.map((m) => (
                <button key={m.id} className="med-result" onClick={() => addMed(m)}>
                  <span className="med-result__name">{medicineName(m, lang)}</span>
                  <span className="med-result__meta">{categoryName(m, lang)} · {m.spec} · {formatMoney(m.price, m.currency)}</span>
                </button>
              ))}
            </div>
          )}

          {rxItems.length === 0 ? (
            <Empty icon="Pill" text={t('consultation.empty')} />
          ) : (
            <div className="rx-edit">
              {rxItems.map((it, idx) => (
                <div key={it.medicine_id} className="rx-edit__row">
                  <div className="rx-edit__info">
                    <strong>{it.name}</strong>
                    <small>{it.spec}</small>
                  </div>
                  <Input
                    type="number"
                    min="1"
                    value={it.quantity}
                    className="rx-edit__qty"
                    onChange={(e) => updateItem(idx, { quantity: Math.max(1, Number(e.target.value) || 1) })}
                  />
                  <Input className="rx-edit__dosage" placeholder={t('consultation.dosage')} value={it.dosage} onChange={(e) => updateItem(idx, { dosage: e.target.value })} />
                  <Input className="rx-edit__freq" placeholder={t('consultation.frequency')} value={it.frequency} onChange={(e) => updateItem(idx, { frequency: e.target.value })} />
                  <button className="icon-btn icon-btn--danger" onClick={() => setRxItems((p) => p.filter((_, i) => i !== idx))} aria-label={t('consultation.remove')}>
                    <Reicon name="Trash" size={16} />
                  </button>
                </div>
              ))}
              <div className="rx-total">
                <span>{t('consultation.total')}</span>
                <strong>{formatMoney(total, currency)}</strong>
              </div>
              <Button variant="primary" icon="Pill" disabled={!patientId || saving} onClick={issueRx} className="btn--block">
                {t('consultation.submitRx')}
              </Button>
            </div>
          )}
        </Card>
      </div>

      <Modal
        open={newOpen}
        onClose={() => setNewOpen(false)}
        title={t('patients.new')}
        footer={
          <>
            <Button variant="ghost" onClick={() => setNewOpen(false)}>{t('common.cancel')}</Button>
            <Button icon="Save" onClick={createPatient}>{t('patients.form.create')}</Button>
          </>
        }
      >
        <div className="form-grid">
          <Field label={t('patients.form.name')}>
            <Input value={newForm.name} onChange={(e) => setNewForm({ ...newForm, name: e.target.value })} />
          </Field>
          <Field label={t('patients.form.gender')}>
            <Select value={newForm.gender} onChange={(e) => setNewForm({ ...newForm, gender: e.target.value })}>
              <option value="M">{t('gender.male')}</option>
              <option value="F">{t('gender.female')}</option>
            </Select>
          </Field>
          <Field label={t('patients.form.birth')}>
            <Input type="date" value={newForm.birth_date} onChange={(e) => setNewForm({ ...newForm, birth_date: e.target.value })} />
          </Field>
          <Field label={t('patients.form.phone')}>
            <Input value={newForm.phone} onChange={(e) => setNewForm({ ...newForm, phone: e.target.value })} />
          </Field>
          <Field label={t('patients.form.idcard')}>
            <Input value={newForm.id_card} onChange={(e) => setNewForm({ ...newForm, id_card: e.target.value })} />
          </Field>
          <Field label={t('patients.form.address')} className="span-2">
            <Input value={newForm.address} onChange={(e) => setNewForm({ ...newForm, address: e.target.value })} />
          </Field>
        </div>
      </Modal>
    </FadeIn>
  );
}
