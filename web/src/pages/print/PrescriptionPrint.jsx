import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import client from '../../api/client';
import { Button } from '../../components/ui';

// Full-screen, print-optimized prescription view. Not wrapped by the app shell so
// the browser print dialog only captures the document.
export default function PrescriptionPrint() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [rx, setRx] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    client.get(`/prescriptions/${id}`).then((r) => setRx(r.data)).finally(() => setLoading(false));
  }, [id]);

  const doPrint = async () => {
    // Record the print job (auto-routes to the doctor/room-bound printer).
    try {
      await client.post('/print-jobs', {
        patient_id: rx.patient_id, doc_type: 'prescription', ref_id: rx.id, summary: `处方 ${rx.prescription_no}`,
      });
    } catch { /* non-fatal */ }
    window.print();
  };

  if (loading) return <div className="print-loading">…</div>;
  if (!rx) return <div className="print-loading">{t('common.empty')}</div>;

  return (
    <div className="print-page">
      <div className="print-toolbar no-print">
        <Button icon="ArrowLeft" variant="ghost" onClick={() => navigate(-1)}>{t('common.back')}</Button>
        <Button icon="Print" onClick={doPrint}>{t('print.printBtn')}</Button>
      </div>
      <div className="rx-sheet">
        <h2 className="rx-title">{t('print.hospitalName')}</h2>
        <h3 className="rx-subtitle">{t('print.prescriptionTitle')}</h3>
        <div className="rx-meta">
          <span>{t('print.rxNo')}: {rx.prescription_no}</span>
          <span>{t('print.date')}: {(rx.created_at || '').slice(0, 10)}</span>
          <span>{t('print.room')}: {rx.room_no || '—'}</span>
        </div>
        <div className="rx-meta">
          <span>{t('print.patient')}: #{rx.patient_id}</span>
          <span>{t('print.prescriber')}: {rx.doctor_name}</span>
        </div>
        <table className="rx-table">
          <thead>
            <tr>
              <th>{t('medicines.name')}</th>
              <th>{t('consultation.dosage')}</th>
              <th>{t('consultation.frequency')}</th>
              <th>{t('consultation.quantity')}</th>
              <th>{t('medicines.price')}</th>
            </tr>
          </thead>
          <tbody>
            {rx.items?.map((it) => (
              <tr key={it.id}>
                <td>{it.medicine_name} <small>{it.spec}</small></td>
                <td>{it.dosage}</td>
                <td>{it.frequency}</td>
                <td>{it.quantity}</td>
                <td>{it.subtotal?.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <div className="rx-total">{t('print.total')}: {rx.total_amount?.toFixed(2)} {rx.currency}</div>
        <div className="rx-sign">
          <span>{t('print.prescriber')}: ____________</span>
          <span>{t('print.date')}: ____________</span>
        </div>
      </div>
    </div>
  );
}
