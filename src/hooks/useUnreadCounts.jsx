import { useEffect, useState } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { emitRealtimeNotification, logRealtimeNotification } from '@/lib/notificationBus';

export default function useUnreadCounts(user) {
    const [counts, setCounts] = useState({ notifications: 0, chats: 0 });

    useEffect(() => {
        if (!user) {
            setCounts({ notifications: 0, chats: 0 });
            return;
        }

        let cancelled = false;
        let ws = null;

        const loadCounts = async () => {
            try {
                const [notifRes, rooms] = await Promise.all([
                    fetch(`${nexusApi.BASE_URL}/notifications/unread-count`, {
                        headers: { Authorization: `Bearer ${localStorage.getItem('nexus_forum_session_token')}` },
                    }).then((r) => r.json()).catch(() => ({ count: 0 })),
                    nexusApi.entities.ChatRoom.filter({ participants: user.id }, '-last_message_at', 100),
                ]);

                if (!cancelled) {
                    const next = {
                        notifications: Number(notifRes?.count) || 0,
                        chats: rooms.reduce((sum, room) => sum + (room.unread_count || 0), 0),
                    };
                    console.info('[nexus:notification] unread count loaded (http)', next);
                    setCounts(next);
                }
            } catch (err) {
                console.error('[nexus:notification] failed to load unread counts:', err);
            }
        };

        loadCounts();

        const token = localStorage.getItem('nexus_forum_session_token');
        if (token) {
            const apiBase = nexusApi.BASE_URL;
            const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            let wsHost;
            if (apiBase.startsWith('http')) {
                try {
                    const url = new URL(apiBase);
                    wsHost = url.host;
                } catch {
                    wsHost = window.location.host;
                }
            } else {
                wsHost = window.location.host;
            }
            const wsUrl = `${wsProtocol}//${wsHost}/api/ws/global`;

            try {
                ws = new WebSocket(wsUrl, ['Bearer', token]);

                ws.onopen = () => {
                    console.info('[nexus:notification] global ws connected', { wsUrl });
                };

                ws.onmessage = (event) => {
                    try {
                        const msg = JSON.parse(event.data);
                        logRealtimeNotification('receive', msg);
                        emitRealtimeNotification(msg);

                        if (msg.type === 'unread_count') {
                            setCounts((prev) => {
                                const next = {
                                    ...prev,
                                    notifications: parseInt(msg.count, 10) || 0,
                                };
                                console.info('[nexus:notification] unread count updated (ws)', {
                                    before: prev.notifications,
                                    after: next.notifications,
                                });
                                return next;
                            });
                        }
                    } catch (err) {
                        console.error('[nexus:notification] failed to parse global ws message:', err);
                    }
                };

                ws.onerror = (err) => {
                    console.error('[nexus:notification] global ws error', err);
                };

                ws.onclose = (ev) => {
                    console.warn('[nexus:notification] global ws closed', { code: ev.code, reason: ev.reason });
                };
            } catch (err) {
                console.error('[nexus:notification] failed to connect global ws:', err);
            }
        }

        const unsubNotifications = nexusApi.entities.Notification.subscribe(loadCounts);
        const unsubChats = nexusApi.entities.ChatRoom.subscribe(loadCounts);

        return () => {
            cancelled = true;
            if (ws) {
                ws.close();
            }
            unsubNotifications();
            unsubChats();
        };
    }, [user]);

    return counts;
}
