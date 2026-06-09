import { useEffect, useState } from 'react';
import { nexusApi } from '@/api/nexusApi';

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
                    setCounts({
                        notifications: Number(notifRes?.count) || 0,
                        chats: rooms.reduce((sum, room) => sum + (room.unread_count || 0), 0),
                    });
                }
            } catch (err) {
                console.error("Failed to load unread counts:", err);
            }
        };

        loadCounts();

        // Connect to global WS
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
                ws = new WebSocket(wsUrl, ["Bearer", token]);
                
                ws.onmessage = (event) => {
                    try {
                        const msg = JSON.parse(event.data);
                        if (msg.type === 'unread_count') {
                            setCounts(prev => ({
                                ...prev,
                                notifications: parseInt(msg.count) || 0
                            }));
                        } else if (msg.type === 'notification') {
                            loadCounts();
                        }
                    } catch (err) {
                        console.error("Failed to parse global ws message:", err);
                    }
                };

                ws.onclose = () => {
                    console.log("Global WebSocket connection closed");
                };
            } catch (err) {
                console.error("Failed to connect to global WS:", err);
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

