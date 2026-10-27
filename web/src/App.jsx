import { Routes, Route, Navigate } from 'react-router-dom';
import { useAuth } from './context/AuthContext';
import AppLayout from './components/layout/AppLayout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Patients from './pages/Patients';
import PatientDetail from './pages/PatientDetail';
import Consultation from './pages/Consultation';
import Medicines from './pages/Medicines';
import Rooms from './pages/Rooms';
import Printers from './pages/Printers';
import Surgeries from './pages/Surgeries';
import Imaging from './pages/Imaging';
import RBAC from './pages/RBAC';
import Operations from './pages/Operations';
import Settings from './pages/Settings';
import PrescriptionPrint from './pages/print/PrescriptionPrint';

function Protected({ children }) {
  const { user } = useAuth();
  if (!user) return <Navigate to="/login" replace />;
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      {/* Printable documents render full-screen, outside the app shell. */}
      <Route path="/print/prescription/:id" element={<PrescriptionPrint />} />
      <Route
        path="/"
        element={
          <Protected>
            <AppLayout />
          </Protected>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<Dashboard />} />
        <Route path="patients" element={<Patients />} />
        <Route path="patients/:id" element={<PatientDetail />} />
        <Route path="consultation" element={<Consultation />} />
        <Route path="medicines" element={<Medicines />} />
        <Route path="rooms" element={<Rooms />} />
        <Route path="printers" element={<Printers />} />
        <Route path="surgeries" element={<Surgeries />} />
        <Route path="imaging" element={<Imaging />} />
        <Route path="rbac" element={<RBAC />} />
        {/* 后台运营（全院核算），受 report:read 权限保护 */}
        <Route path="operations" element={<Operations />} />
        <Route path="settings" element={<Settings />} />
      </Route>
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  );
}
