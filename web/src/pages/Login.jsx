import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../context/AuthContext';
import { Button, Input, Field } from '../components/ui';
import Reicon from '../components/Reicon';
import AnimatedBackground from '../components/anim/AnimatedBackground';
import BlurText from '../components/anim/BlurText';

export default function Login() {
  const { t } = useTranslation();
  const { login, loading } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ username: 'doctor', password: 'doctor123' });
  const [error, setError] = useState('');

  const submit = async (e) => {
    e.preventDefault();
    setError('');
    const res = await login(form.username, form.password);
    if (res.ok) navigate('/dashboard');
    else setError(t('login.error'));
  };

  return (
    <div className="login-page">
      <AnimatedBackground />
      <div className="login-card">
        <div className="login-brand">
          <span className="login-logo">
            <Reicon name="Heart" size={26} color="#fff" />
          </span>
          <BlurText text={t('app.name')} className="login-brand__title" />
          <p>{t('app.tagline')}</p>
        </div>
        <form onSubmit={submit} className="login-form">
          {error && <div className="alert alert--error">{error}</div>}
          <Field label={t('login.username')}>
            <Input
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              placeholder="admin / doctor"
            />
          </Field>
          <Field label={t('login.password')}>
            <Input
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="••••••"
            />
          </Field>
          <Button type="submit" icon="Lock" disabled={loading} className="btn--block">
            {loading ? t('common.loading') : t('login.submit')}
          </Button>
          <div className="login-hint">
            <span>
              <Reicon name="AlertTriangle" size={14} /> {t('login.demoAdmin')}
            </span>
            <span>
              <Reicon name="AlertTriangle" size={14} /> {t('login.demoDoctor')}
            </span>
          </div>
        </form>
      </div>
    </div>
  );
}
