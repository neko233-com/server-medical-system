import { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../api/client';
import { Card, Button, Input, Field, Table, Modal, Spinner, Badge, Select } from '../components/ui';
import { useAuth } from '../context/AuthContext';
import FadeIn from '../components/anim/FadeIn';

const GROUP_MAP = {
  仪表盘: 'dashboard', 患者: 'patient', 病例: 'case', 处方: 'prescription', 药品: 'medicine',
  手术: 'surgery', 影像: 'imaging', 房间: 'room', 打印机: 'printer', 账号: 'user', 角色: 'role',
};

export default function RBAC() {
  const { t } = useTranslation();
  const { can, user } = useAuth();
  const [tab, setTab] = useState('users');
  const [users, setUsers] = useState([]);
  const [roles, setRoles] = useState([]);
  const [perms, setPerms] = useState([]);
  const [loading, setLoading] = useState(true);
  const [userModal, setUserModal] = useState(false);
  const [roleModal, setRoleModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState(null);
  const [rolePerms, setRolePerms] = useState({});
  const [uform, setUform] = useState({ id: 0, username: '', name: '', password: '', role: 'doctor', department: '', title: '', status: 'active' });
  const [rform, setRform] = useState({ code: '', name_zh: '', description: '' });

  const loadRolesPerms = useCallback(() => {
    client.get('/rbac/roles').then((r) => setRoles(r.data)).catch(() => {});
    client.get('/rbac/permissions').then((r) => setPerms(r.data)).catch(() => {});
  }, []);

  useEffect(() => { loadRolesPerms(); }, [loadRolesPerms]);

  useEffect(() => {
    if (tab === 'users') {
      setLoading(true);
      client.get('/rbac/users').then((r) => setUsers(r.data)).finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, [tab]);

  const openUser = (u) => setUform({
    id: u.id, username: u.username, name: u.name, password: '', role: u.role,
    department: u.department || '', title: u.title || '', status: u.status || 'active',
  });

  const saveUser = async () => {
    const payload = { ...uform, role: uform.role };
    if (uform.id) {
      await client.put(`/rbac/users/${uform.id}`, payload);
    } else {
      await client.post('/rbac/users', payload);
    }
    setUserModal(false);
    setUform({ id: 0, username: '', name: '', password: '', role: 'doctor', department: '', title: '', status: 'active' });
    const r = await client.get('/rbac/users'); setUsers(r.data);
  };

  const toggleUser = async (u) => {
    await client.put(`/rbac/users/${u.id}`, { status: u.status === 'active' ? 'disabled' : 'active' });
    const r = await client.get('/rbac/users'); setUsers(r.data);
  };

  const assignRole = async (u) => {
    const code = window.prompt(t('rbac.users.role') + ' (code)', u.role);
    if (!code) return;
    await client.put(`/rbac/users/${u.id}/role`, { role: code });
    const r = await client.get('/rbac/users'); setUsers(r.data);
  };

  const selectRole = (role) => {
    setSelectedRole(role);
    const set = {};
    (role.permissions || []).forEach((p) => { set[p] = true; });
    setRolePerms(set);
  };

  const saveRolePerms = async () => {
    if (!selectedRole) return;
    const codes = Object.keys(rolePerms).filter((k) => rolePerms[k]);
    await client.put(`/rbac/roles/${selectedRole.code}/permissions`, { permissions: codes });
    loadRolesPerms();
    selectRole({ ...selectedRole, permissions: codes });
  };

  const createRole = async () => {
    if (!rform.code || !rform.name_zh) return;
    await client.post('/rbac/roles', rform);
    setRoleModal(false);
    setRform({ code: '', name_zh: '', description: '' });
    loadRolesPerms();
  };

  const deleteRole = async (role) => {
    if (!window.confirm(t('rbac.roles.isSystem') + '?')) return;
    await client.delete(`/rbac/roles/${role.code}`);
    loadRolesPerms();
    if (selectedRole?.code === role.code) setSelectedRole(null);
  };

  const roleName = (code) => roles.find((r) => r.code === code)?.name_zh || code;
  const grouped = perms.reduce((acc, p) => { (acc[p.group] = acc[p.group] || []).push(p); return acc; }, {});

  const userColumns = [
    { key: 'name', label: t('rbac.users.table.name'), render: (u) => <strong>{u.name}</strong> },
    { key: 'username', label: t('rbac.users.table.username') },
    { key: 'role', label: t('rbac.users.table.role'), render: (u) => <Badge tone="info">{roleName(u.role)}</Badge> },
    { key: 'department', label: t('rbac.users.table.department') },
    { key: 'title', label: t('rbac.users.table.title') },
    { key: 'status', label: t('rbac.users.table.status'), render: (u) => <Badge tone={u.status === 'active' ? 'success' : 'muted'}>{t('status.' + (u.status === 'active' ? 'active' : 'inactive'))}</Badge> },
    {
      key: 'actions', label: t('common.actions'), align: 'right',
      render: (u) => (
        <div className="row-actions">
          <Button size="sm" variant="ghost" icon="Shield" onClick={() => assignRole(u)}>{t('rbac.users.role')}</Button>
          <Button size="sm" variant="ghost" icon="Edit" onClick={() => openUser(u)} />
          {u.id !== user?.id && <Button size="sm" variant="ghost" icon={u.status === 'active' ? 'Lock' : 'Check'} onClick={() => toggleUser(u)} />}
        </div>
      ),
    },
  ];

  return (
    <FadeIn>
      <div className="toolbar">
        <div className="tabs">
          <button className={tab === 'users' ? 'tab is-active' : 'tab'} onClick={() => setTab('users')}>{t('rbac.users')}</button>
          <button className={tab === 'roles' ? 'tab is-active' : 'tab'} onClick={() => setTab('roles')}>{t('rbac.roles')}</button>
        </div>
        {tab === 'users' && can('user:manage') && <Button icon="Plus" onClick={() => { setUform({ id: 0, username: '', name: '', password: '', role: 'doctor', department: '', title: '', status: 'active' }); setUserModal(true); }}>{t('rbac.users.new')}</Button>}
        {tab === 'roles' && can('role:manage') && <Button icon="Plus" onClick={() => setRoleModal(true)}>{t('rbac.roles.permissions')}</Button>}
      </div>

      {tab === 'users' && (
        <Card padding={false}>
          {loading ? <div className="center-pad"><Spinner /></div> : <Table columns={userColumns} rows={users} empty={t('common.empty')} />}
          <div className="hint-box">{t('rbac.users.demoHint')}</div>
        </Card>
      )}

      {tab === 'roles' && (
        <div className="rbac-roles">
          <Card title={t('rbac.roles')}>
            <div className="role-list">
              {roles.map((r) => (
                <div key={r.code} className={`role-chip ${selectedRole?.code === r.code ? 'is-active' : ''}`} onClick={() => selectRole(r)}>
                  <div><strong>{r.name_zh}</strong> <small>{r.code}</small></div>
                  <span className="muted">{r.permissions?.length || 0} {t('rbac.roles.permissions')}</span>
                  {can('role:manage') && !r.is_system && <Button size="sm" variant="danger" icon="Trash" onClick={(e) => { e.stopPropagation(); deleteRole(r); }} />}
                </div>
              ))}
            </div>
          </Card>

          {selectedRole && (
            <Card title={`${selectedRole.name_zh} · ${t('rbac.roles.permissions')}`} actions={
              can('role:manage') && <Button icon="Save" onClick={saveRolePerms}>{t('rbac.roles.savePerms')}</Button>
            }>
              <div className="perm-matrix">
                {Object.entries(grouped).map(([group, list]) => (
                  <div key={group} className="perm-group">
                    <h4>{t('rbac.group.' + (GROUP_MAP[group] || group))}</h4>
                    <div className="perm-items">
                      {list.map((p) => (
                        <label key={p.code} className="perm-item">
                          <input type="checkbox" checked={!!rolePerms[p.code]} disabled={!can('role:manage')} onChange={(e) => setRolePerms({ ...rolePerms, [p.code]: e.target.checked })} />
                          <span>{localStorage.getItem('app_lang') === 'en-US' ? p.name_en : p.name_zh}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </Card>
          )}
        </div>
      )}

      <Modal open={userModal} onClose={() => setUserModal(false)} title={uform.id ? t('rbac.users.table.name') : t('rbac.users.new')} footer={
        <>
          <Button variant="ghost" onClick={() => setUserModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={saveUser}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label={t('rbac.users.username')}><Input value={uform.username} disabled={!!uform.id} onChange={(e) => setUform({ ...uform, username: e.target.value })} /></Field>
          <Field label={t('rbac.users.name')}><Input value={uform.name} onChange={(e) => setUform({ ...uform, name: e.target.value })} /></Field>
          {!uform.id && <Field label={t('rbac.users.password')}><Input type="password" value={uform.password} onChange={(e) => setUform({ ...uform, password: e.target.value })} placeholder="••••••" /></Field>}
          <Field label={t('rbac.users.role')}>
            <Select value={uform.role} onChange={(e) => setUform({ ...uform, role: e.target.value })}>{roles.map((r) => <option key={r.code} value={r.code}>{r.name_zh}</option>)}</Select>
          </Field>
          <Field label={t('rbac.users.department')}><Input value={uform.department} onChange={(e) => setUform({ ...uform, department: e.target.value })} /></Field>
          <Field label={t('rbac.users.title')}><Input value={uform.title} onChange={(e) => setUform({ ...uform, title: e.target.value })} /></Field>
          <Field label={t('rbac.users.status')}>
            <Select value={uform.status} onChange={(e) => setUform({ ...uform, status: e.target.value })}><option value="active">{t('status.active')}</option><option value="disabled">{t('status.inactive')}</option></Select>
          </Field>
        </div>
      </Modal>

      <Modal open={roleModal} onClose={() => setRoleModal(false)} title={t('rbac.roles.permissions')} footer={
        <>
          <Button variant="ghost" onClick={() => setRoleModal(false)}>{t('common.cancel')}</Button>
          <Button icon="Save" onClick={createRole}>{t('common.save')}</Button>
        </>
      }>
        <div className="form-grid">
          <Field label="Code"><Input value={rform.code} onChange={(e) => setRform({ ...rform, code: e.target.value })} placeholder="custom_role" /></Field>
          <Field label={t('rbac.users.table.name')}><Input value={rform.name_zh} onChange={(e) => setRform({ ...rform, name_zh: e.target.value })} /></Field>
          <Field label={t('rbac.roles.description')} className="span-2"><Input value={rform.description} onChange={(e) => setRform({ ...rform, description: e.target.value })} /></Field>
        </div>
      </Modal>
    </FadeIn>
  );
}
