import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Spinner, Empty, Badge } from '../components/ui';
import Reicon from '../components/Reicon';
import FadeIn from '../components/anim/FadeIn';
import { genderLabel, formatMoney } from '../utils/format';

export default function PatientDetail() {
  const { id } = useParams();
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const navigate = useNavigate();
  const [patient, setPatient] = useState(null);
  const [cases, setCases] = useState([]);
  const [rx, setRx] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      client.get(`/patients/${id}`),
      client.get('/cases', { params: { patient_id: id, page_size: 50 } }),
      client.get('/prescriptions', { params: { patient_id: id, page_size: 50 } }),
    ])
      .then(([p, c, r]) => {
        setPatient(p.data);
        setCases(c.data.items || []);
        setRx(r.data.items || []);
      })
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return <div className="center-pad"><Spinner /></div>;
  if (!patient) return <Empty text={t('patients.empty')} />;

  return (
    <FadeIn className="stack-lg">
      <div className="detail-head">
        <button className="icon-btn" onClick={() => navigate('/patients')} aria-label={t('common.back')}>
          <Reicon name="ArrowLeft" size={18} />
        </button>
        <span className="avatar avatar--lg">{(patient.name || '?').slice(0, 1)}</span>
        <div className="detail-head__meta">
          <h2>{patient.name}</h2>
          <p>
            {genderLabel(patient.gender, t)} · {patient.birth_date || '—'} · {patient.phone || '—'}
          </p>
          {patient.address && <p className="detail-head__addr">{patient.address}</p>}
        </div>
        <div className="detail-head__actions">
          <Button icon="Stethoscope" onClick={() => navigate(`/consultation?patient=${patient.id}`)}>
            {t('nav.consultation')}
          </Button>
        </div>
      </div>

      <Card title={t('nav.consultation')}>
        {cases.length === 0 ? (
          <Empty icon="Note" text={t('dashboard.noRecent')} />
        ) : (
          <div className="timeline">
            {cases.map((c) => (
              <div key={c.id} className="timeline__item">
                <div className="timeline__date">{(c.visit_date || '').slice(0, 10)}</div>
                <div className="timeline__body">
                  <div className="timeline__row">
                    <span className="tag tag--muted">{t('consultation.chiefComplaint')}</span>
                    <span>{c.chief_complaint || '—'}</span>
                  </div>
                  <div className="timeline__row">
                    <span className="tag tag--muted">{t('consultation.diagnosis')}</span>
                    <span>{c.diagnosis || '—'}</span>
                  </div>
                  {c.notes && (
                    <div className="timeline__row">
                      <span className="tag tag--muted">{t('consultation.notes')}</span>
                      <span>{c.notes}</span>
                    </div>
                  )}
                  <small className="timeline__doc">{c.doctor_name} · {c.case_no}</small>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card title={t('medicines.title').split('（')[0]}>
        {rx.length === 0 ? (
          <Empty icon="Pill" text={t('consultation.empty')} />
        ) : (
          <div className="rx-list">
            {rx.map((r) => (
              <div key={r.id} className="rx-card">
                <div className="rx-card__head">
                  <strong>{r.prescription_no}</strong>
                  <Badge tone="teal">{formatMoney(r.total_amount, r.currency)}</Badge>
                </div>
                <ul className="rx-card__items">
                  {r.items?.map((it) => (
                    <li key={it.id}>
                      <span>{it.medicine_name}</span>
                      <small>x{it.quantity} · {it.dosage || '—'} · {it.frequency || '—'}</small>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}
      </Card>
    </FadeIn>
  );
}
