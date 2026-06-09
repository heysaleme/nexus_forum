import { useState, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { Bell, MessageCircle, Heart, UserPlus, Shield, Star, Check, CheckCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { motion, AnimatePresence } from 'framer-motion';
import { formatDistanceToNow } from 'date-fns';
import { ru } from 'date-fns/locale';
import { Link } from 'react-router-dom';
import { onRealtimeNotification, logRealtimeNotification } from '@/lib/notificationBus';

const NOTIF_ICONS = {
    reply: { icon: MessageCircle, color: 'bg-blue-100 text-blue-600' },
    comment: { icon: MessageCircle, color: 'bg-blue-100 text-blue-600' },
    mention: { icon: Star, color: 'bg-yellow-100 text-yellow-600' },
    content_removed: { icon: Shield, color: 'bg-red-100 text-red-600' },
    report_resolved: { icon: Shield, color: 'bg-green-100 text-green-600' },
    follow: { icon: UserPlus, color: 'bg-green-100 text-green-600' },
    message: { icon: MessageCircle, color: 'bg-primary/10 text-primary' },
    moderation: { icon: Shield, color: 'bg-red-100 text-red-600' },
    achievement: { icon: Star, color: 'bg-purple-100 text-purple-600' },
    vote: { icon: Heart, color: 'bg-pink-100 text-pink-600' },
    recommendation: { icon: Bell, color: 'bg-orange-100 text-orange-600' },
};

const FILTERS = ['all', 'reply', 'mention', 'follow', 'message', 'achievement'];

export default function Notifications() {
    const { user } = useAuth();
    const [notifications, setNotifications] = useState([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState('all');

    useEffect(() => {
        if (user) loadNotifications();
    }, [user]);

    useEffect(() => {
        if (!user) return undefined;

        return onRealtimeNotification((msg) => {
            if (msg.type === 'notification' && msg.data) {
                logRealtimeNotification('notifications-page', msg);
                setNotifications((prev) => {
                    const id = Number(msg.data.id);
                    if (prev.some((n) => Number(n.id) === id)) {
                        return prev;
                    }
                    return [msg.data, ...prev];
                });
            }
            if (msg.type === 'unread_count') {
                // Badge on this page derives from notifications state; markRead handles local state.
            }
        });
    }, [user]);

    const loadNotifications = async () => {
        setLoading(true);
        const data = await nexusApi.entities.Notification.list();
        setNotifications(data);
        setLoading(false);
    };

    const markAllRead = async () => {
        const unread = notifications.filter(n => !n.is_read);
        await nexusApi.entities.Notification.readAll();
        setNotifications(prev => prev.map(n => ({ ...n, is_read: true })));
    };

    const markRead = async (notif) => {
        if (notif.is_read) return;
        await nexusApi.entities.Notification.markRead(notif.id);
        setNotifications(prev => prev.map(n => n.id === notif.id ? { ...n, is_read: true } : n));
    };

    const filtered = filter === 'all' ? notifications : notifications.filter(n => n.type === filter);
    const unreadCount = notifications.filter(n => !n.is_read).length;

    return (
        <div className="max-w-2xl mx-auto px-4 py-4">
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                    <Bell className="w-5 h-5 text-primary" />
                    <h1 className="text-xl font-display font-black">Уведомления</h1>
                    {unreadCount > 0 && (
                        <Badge className="nexus-gradient border-0 text-white text-xs">{unreadCount}</Badge>
                    )}
                </div>
                {unreadCount > 0 && (
                    <Button variant="ghost" size="sm" onClick={markAllRead} className="rounded-xl gap-1.5 text-xs">
                        <CheckCheck className="w-3.5 h-3.5" />
                        Прочитать все
                    </Button>
                )}
            </div>

            {/* Filters */}
            <div className="flex gap-2 overflow-x-auto scrollbar-hide mb-4 pb-1">
                {FILTERS.map(f => (
                    <button
                        key={f}
                        onClick={() => setFilter(f)}
                        className={`px-3 py-1.5 rounded-xl text-xs font-semibold whitespace-nowrap flex-shrink-0 transition-all ${filter === f ? 'nexus-gradient text-white shadow-nexus' : 'bg-muted/60 text-muted-foreground'
                            }`}
                    >
                        {f === 'all' ? 'Все' : f === 'reply' ? 'Ответы' : f === 'mention' ? 'Упоминания' : f === 'follow' ? 'Подписки' : f === 'message' ? 'Сообщения' : f === 'achievement' ? 'Достижения' : f}
                    </button>
                ))}
            </div>

            {loading ? (
                <LoadingSpinner size="lg" className="py-20" />
            ) : filtered.length === 0 ? (
                <EmptyState icon={Bell} title="Уведомлений нет" description="Здесь появятся ответы, упоминания и другие события" />
            ) : (
                <div className="space-y-2">
                    <AnimatePresence>
                        {filtered.map(notif => {
                            const config = NOTIF_ICONS[notif.type] || NOTIF_ICONS.recommendation;
                            const Icon = config.icon;
                            const timeAgo = notif.created_date
                                ? formatDistanceToNow(new Date(notif.created_date), { addSuffix: true, locale: ru })
                                : '';

                            return (
                                <motion.div
                                    key={notif.id}
                                    initial={{ opacity: 0, y: 8 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    exit={{ opacity: 0, y: -8 }}
                                    onClick={() => markRead(notif)}
                                    className={`nexus-card p-4 flex items-start gap-3 cursor-pointer transition-all ${!notif.is_read ? 'ring-2 ring-primary/20 bg-primary/5' : 'hover:bg-muted/30'}`}
                                >
                                    <div className={`w-9 h-9 rounded-2xl flex items-center justify-center flex-shrink-0 ${config.color}`}>
                                        <Icon className="w-4 h-4" />
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        {notif.actor_avatar && (
                                            <img src={notif.actor_avatar} className="w-6 h-6 rounded-full float-right ml-2" alt="" />
                                        )}
                                        {notif.title && <p className="text-sm font-bold text-foreground mb-0.5">{notif.title}</p>}
                                        <p className="text-xs text-muted-foreground leading-relaxed">{notif.body}</p>
                                        <p className="text-[10px] text-muted-foreground mt-1.5">{timeAgo}</p>
                                    </div>
                                    {!notif.is_read && (
                                        <div className="w-2 h-2 bg-primary rounded-full flex-shrink-0 mt-2" />
                                    )}
                                </motion.div>
                            );
                        })}
                    </AnimatePresence>
                </div>
            )}
        </div>
    );
}