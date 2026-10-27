import { createContext, useContext, useEffect, useState, useCallback } from 'react';
import client from '../api/client';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem('user') || 'null');
    } catch {
      return null;
    }
  });
  const [permissions, setPermissions] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem('permissions') || 'null');
    } catch {
      return null;
    }
  });
  const [loading, setLoading] = useState(false);

  const apply = (data) => {
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
    localStorage.setItem('permissions', JSON.stringify(data.permissions || {}));
    setUser(data.user);
    setPermissions(data.permissions || {});
  };

  const login = async (username, password) => {
    setLoading(true);
    try {
      const { data } = await client.post('/auth/login', { username, password });
      apply(data);
      return { ok: true };
    } catch (e) {
      return { ok: false, error: e.response?.data?.error || 'error' };
    } finally {
      setLoading(false);
    }
  };

  const logout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('permissions');
    setUser(null);
    setPermissions(null);
  };

  // Refresh the profile if a token exists (e.g. after a page reload).
  useEffect(() => {
    if (user) return;
    const token = localStorage.getItem('token');
    if (!token) return;
    client
      .get('/auth/me')
      .then(({ data }) => {
        localStorage.setItem('user', JSON.stringify(data.user));
        localStorage.setItem('permissions', JSON.stringify(data.permissions || {}));
        setUser(data.user);
        setPermissions(data.permissions || {});
      })
      .catch(() => logout());
  }, [user]);

  // can checks whether the current user holds the given permission code.
  const can = useCallback(
    (code) => {
      if (!permissions) return false;
      return Boolean(permissions[code]);
    },
    [permissions]
  );

  return (
    <AuthContext.Provider value={{ user, permissions, setUser, setPermissions, login, logout, loading, can }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);
