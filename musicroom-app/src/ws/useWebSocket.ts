import { useEffect, useRef } from 'react';
import { getBaseUrl } from '../storage/storage';
import { eventsApi } from '../api/events';

type Msg = { type: string; [k: string]: unknown };

export function useWebSocket(
  room: 'events' | 'playlists',
  id: string,
  onMessage: (msg: Msg) => void,
) {
  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    connect();
    return () => {
      mountedRef.current = false;
      retryRef.current && clearTimeout(retryRef.current);
      wsRef.current?.close();
    };
  }, [room, id]);

  async function connect() {
    try {
      const [baseUrl, { ticket }] = await Promise.all([
        getBaseUrl(),
        eventsApi.wsTicket(),
      ]);
      if (!mountedRef.current) return;
      const wsUrl = baseUrl
        .replace(/^http/, 'ws')
        .replace(/^https/, 'wss');
      const ws = new WebSocket(`${wsUrl}/ws/${room}/${id}?ticket=${ticket}`);
      wsRef.current = ws;
      ws.onmessage = (e) => {
        try { onMessage(JSON.parse(e.data)); } catch {}
      };
      ws.onclose = () => {
        if (!mountedRef.current) return;
        retryRef.current = setTimeout(connect, 2000);
      };
    } catch {
      if (!mountedRef.current) return;
      retryRef.current = setTimeout(connect, 3000);
    }
  }
}
