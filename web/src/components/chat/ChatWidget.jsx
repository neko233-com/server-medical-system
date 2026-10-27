import { useEffect, useRef, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import client from '../../api/client';
import { useAuth } from '../../context/AuthContext';
import Reicon from '../Reicon';

// ---------------------------------------------------------------------------
//  院内即时通讯浮窗
// ---------------------------------------------------------------------------
//  能力：
//   • 同事互相私聊 + 全院大群
//   • 文字 / 图片消息
//   • WebSocket 实时投递，断线自动重连（指数退避上限 8s）
//   • 在线名单实时更新（presence 帧）
//   • 未读角标（窗口关闭时累计）
//  历史由 REST 拉取，实时增量由 WS 推送，二者按 id 去重。
// ---------------------------------------------------------------------------

// 聊天图片同样走鉴权接口，<img> 无法带 Bearer，故以 blob 方式加载。
function ChatImage({ fileName, origName }) {
  const [url, setUrl] = useState(null);
  useEffect(() => {
    let revoked = false;
    let objectUrl = null;
    client
      .get(`/chat/files/${fileName}`, { responseType: 'blob' })
      .then((r) => {
        if (revoked) return;
        objectUrl = URL.createObjectURL(r.data);
        setUrl(objectUrl);
      })
      .catch(() => {});
    return () => {
      revoked = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [fileName]);
  if (!url) return <div className="chat-img chat-img--loading" />;
  // eslint-disable-next-line jsx-a11y/alt-text
  return <img src={url} className="chat-img" alt={origName || ''} onClick={() => window.open(url, '_blank')} />;
}

// wsUrl 自动适配 http/https，并与后端同源。
function wsUrl(token) {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/api/chat/ws?token=${encodeURIComponent(token)}`;
}

export default function ChatWidget() {
  const { t } = useTranslation();
  const { user } = useAuth();

  const [open, setOpen] = useState(false);
  const [connected, setConnected] = useState(false);
  const [online, setOnline] = useState([]);
  const [messages, setMessages] = useState([]);
  const [peers, setPeers] = useState([]); // full staff directory for picking DM targets
  const [mode, setMode] = useState('group'); // group | direct
  const [peer, setPeer] = useState(null);
  const [draft, setDraft] = useState('');
  const [unread, setUnread] = useState(0);
  const [uploading, setUploading] = useState(false);

  const wsRef = useRef(null);
  const listRef = useRef(null);
  const fileRef = useRef(null);
  const retryRef = useRef(0);
  const timerRef = useRef(null);

  // 当前会话的消息需剔除另一端的记录：按 scope/key 过滤
  const visible = messages.filter((m) => {
    if (mode === 'group') return m.scope === 'group';
    if (!peer) return false;
    const a = Math.min(user.id, peer.user_id);
    const b = Math.max(user.id, peer.user_id);
    return m.scope === 'direct' && m.peer_key === `${a}-${b}`;
  });

  // ---- 拉取当前会话历史 ----
  const loadHistory = useCallback(() => {
    if (!user) return;
    const params = { limit: 50 };
    if (mode === 'group') params.group = '1';
    else if (peer) params.peer = peer.user_id;
    else return;
    client
      .get('/chat/history', { params })
      .then((r) => setMessages(r.data.items || []))
      .catch(() => {});
  }, [user, mode, peer]);

  // ---- 人员字典（用于发起私聊）与在线名单 ----
  useEffect(() => {
    if (!open || !user) return;
    client.get('/rbac/directory').then((r) => setPeers(r.data || [])).catch(() => {});
    client.get('/chat/online').then((r) => setOnline(r.data.items || [])).catch(() => {});
  }, [open, user]);

  useEffect(() => {
    if (open) loadHistory();
  }, [open, loadHistory]);

  useEffect(() => {
    if (open) {
      setUnread(0);
      // 滚到底
      requestAnimationFrame(() => {
        if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight;
      });
    }
  }, [open, visible.length]);

  // ---- WebSocket 生命周期 ----
  useEffect(() => {
    if (!user) return;
    const token = localStorage.getItem('token');
    if (!token) return;

    let closedByUs = false;
    const connect = () => {
      const ws = new WebSocket(wsUrl(token));
      wsRef.current = ws;
      ws.onopen = () => {
        setConnected(true);
        retryRef.current = 0;
      };
      ws.onclose = () => {
        setConnected(false);
        if (closedByUs) return;
        // 指数退避重连，避免服务端抖动时疯狂重连
        retryRef.current = Math.min(retryRef.current + 1, 5);
        const delay = Math.min(1000 * 2 ** retryRef.current, 8000);
        timerRef.current = setTimeout(connect, delay);
      };
      ws.onmessage = (ev) => {
        let frame;
        try {
          frame = JSON.parse(ev.data);
        } catch {
          return;
        }
        if (frame.type === 'presence' || frame.type === 'online') {
          setOnline(frame.online || []);
          return;
        }
        if (frame.type === 'message') {
          setMessages((prev) => {
            // 幂等去重：REST 历史与 WS 推送可能重合
            if (prev.some((m) => m.id === frame.id)) return prev;
            return [...prev, frame];
          });
          if (!open) setUnread((n) => n + 1);
        }
      };
    };
    connect();

    return () => {
      closedByUs = true;
      clearTimeout(timerRef.current);
      if (wsRef.current) wsRef.current.close();
    };
  }, [user, open]);

  // ---- 发送 ----
  const send = (payload) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      // 未连接时给出提示，避免用户误以为已发送
      setMessages((prev) => [
        ...prev,
        { id: `tmp-${Date.now()}`, scope: mode, content: t('chat.notConnected'), content_type: 'text' },
      ]);
      return;
    }
    ws.send(JSON.stringify(payload));
  };

  const sendText = () => {
    const text = draft.trim();
    if (!text) return;
    const base = { type: 'message', content_type: 'text', content: text };
    if (mode === 'group') send({ ...base, scope: 'group' });
    else if (peer) send({ ...base, scope: 'direct', receiver_id: peer.user_id });
    setDraft('');
  };

  const pickImage = async (e) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;
    setUploading(true);
    try {
      const fd = new FormData();
      fd.append('file', file);
      const r = await client.post('/chat/images', fd, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      const base = {
        type: 'message',
        content_type: 'image',
        content: '',
        file_name: r.data.file_name,
        orig_name: r.data.orig_name,
        file_size: r.data.file_size,
      };
      if (mode === 'group') send({ ...base, scope: 'group' });
      else if (peer) send({ ...base, scope: 'direct', receiver_id: peer.user_id });
    } catch {
      /* 上传失败静默处理，界面保持可用 */
    } finally {
      setUploading(false);
    }
  };

  const isOnline = (uid) => online.some((o) => o.user_id === uid);
  const others = peers.filter((p) => p.user_id !== user?.id);

  return (
    <div className="chat-widget">
      {open && (
        <div className="chat-panel">
          <header className="chat-panel__head">
            <div className="chat-tabs">
              <button
                className={`chat-tab ${mode === 'group' ? 'is-active' : ''}`}
                onClick={() => {
                  setMode('group');
                  setPeer(null);
                }}
              >
                <Reicon name="Users" size={16} /> {t('chat.hospitalGroup')}
              </button>
              <button className={`chat-tab ${mode === 'direct' ? 'is-active' : ''}`} onClick={() => setMode('direct')}>
                <Reicon name="MessageCircle" size={16} /> {t('chat.direct')}
              </button>
            </div>
            <span className={`chat-dot ${connected ? 'is-on' : ''}`} title={connected ? t('chat.connected') : t('chat.connecting')} />
            <button className="chat-icon-btn" onClick={() => setOpen(false)} aria-label="close">
              <Reicon name="X" size={18} />
            </button>
          </header>

          <div className="chat-body">
            {mode === 'direct' && (
              <aside className="chat-contacts">
                {others.map((p) => (
                  <button
                    key={p.user_id}
                    className={`contact ${peer?.user_id === p.user_id ? 'is-active' : ''}`}
                    onClick={() => setPeer(p)}
                  >
                    <span className="avatar avatar--sm">{p.name ? p.name.slice(0, 1) : '?'}</span>
                    <span className="contact__meta">
                      <strong>{p.name}</strong>
                      <small>{p.department || p.role}</small>
                    </span>
                    <span className={`presence-dot ${isOnline(p.user_id) ? 'is-on' : ''}`} />
                  </button>
                ))}
              </aside>
            )}

            <section className="chat-thread">
              <div className="chat-msgs" ref={listRef}>
                {visible.length === 0 && <div className="chat-empty">{t('chat.noMessages')}</div>}
                {mode === 'direct' && peer === null && (
                  <div className="chat-empty">{t('chat.pickPeer')}</div>
                )}
                {visible.map((m) => {
                  const mine = m.sender_id === user?.id;
                  return (
                    <div key={m.id} className={`bubble-row ${mine ? 'is-mine' : ''}`}>
                      {!mine && <span className="avatar avatar--sm">{(m.sender_name || '?').slice(0, 1)}</span>}
                      <div className="bubble">
                        {!mine && <small className="bubble__from">{m.sender_name}</small>}
                        {m.content_type === 'image' && m.file_name ? (
                          <ChatImage fileName={m.file_name} origName={m.orig_name} />
                        ) : (
                          <p className="bubble__text">{m.content}</p>
                        )}
                        <time className="bubble__time">
                          {m.created_at ? new Date(m.created_at).toLocaleString(undefined, { hour12: false }) : ''}
                        </time>
                      </div>
                    </div>
                  );
                })}
              </div>

              <footer className="chat-composer">
                <input type="file" accept="image/*" ref={fileRef} hidden onChange={pickImage} />
                <button className="chat-icon-btn" onClick={() => fileRef.current?.click()} disabled={uploading} title={t('chat.sendImage')}>
                  <Reicon name={uploading ? 'Refresh' : 'Image'} size={18} />
                </button>
                <input
                  className="chat-input"
                  value={draft}
                  placeholder={mode === 'group' ? t('chat.groupPlaceholder') : t('chat.dmPlaceholder')}
                  onChange={(e) => setDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') sendText();
                  }}
                  disabled={mode === 'direct' && !peer}
                />
                <button className="chat-icon-btn chat-icon-btn--send" onClick={sendText} title={t('chat.send')}>
                  <Reicon name="Send" size={18} />
                </button>
              </footer>
            </section>
          </div>
        </div>
      )}

      <button className="chat-fab" onClick={() => setOpen((v) => !v)} title={t('chat.title')}>
        <Reicon name={open ? 'X' : 'MessageCircle'} size={22} />
        {!open && unread > 0 && <span className="chat-badge">{unread > 99 ? '99+' : unread}</span>}
      </button>
    </div>
  );
}
