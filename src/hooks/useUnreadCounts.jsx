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

        const loadCounts = async () => {
            const [notifications, rooms] = await Promise.all([
                nexusApi.entities.Notification.filter({ user_id: user.id }, '-created_date', 100),
                nexusApi.entities.ChatRoom.filter({ participants: user.id }, '-last_message_at', 100),
            ]);

            if (!cancelled) {
                setCounts({
                    notifications: notifications.filter((item) => !item.is_read).length,
                    chats: rooms.reduce((sum, room) => sum + (room.unread_count || 0), 0),
                });
            }
        };

        loadCounts();

        const unsubNotifications = nexusApi.entities.Notification.subscribe(loadCounts);
        const unsubChats = nexusApi.entities.ChatRoom.subscribe(loadCounts);

        return () => {
            cancelled = true;
            unsubNotifications();
            unsubChats();
        };
    }, [user]);

    return counts;
}
