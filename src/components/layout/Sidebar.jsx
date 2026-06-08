import { Link, useLocation } from 'react-router-dom';
import { Home, Compass, PlusCircle, MessageCircle, User, Bell, Settings, Shield, Zap } from 'lucide-react';
import { motion } from 'framer-motion';
import { Badge } from '@/components/ui/badge';
import useUnreadCounts from '@/hooks/useUnreadCounts';
import { useAuth } from '@/lib/AuthContext';

const navItems = [
    { icon: Home, label: 'Лента', path: '/' },
    { icon: Compass, label: 'Сообщества', path: '/communities' },
    { icon: PlusCircle, label: 'Создать', path: '/create' },
    { icon: MessageCircle, label: 'Чаты', path: '/chats' },
    { icon: Bell, label: 'Уведомления', path: '/notifications', badge: 3 },
    { icon: User, label: 'Профиль', path: '/profile' },
    { icon: Settings, label: 'Настройки', path: '/settings' },
];

export default function Sidebar({ user }) {
    const location = useLocation();
    const { triggerAuthModal } = useAuth();
    const { notifications, chats } = useUnreadCounts(user);

    return (
        <aside className="hidden md:flex flex-col w-64 h-screen sticky top-0 bg-card border-r border-border/50 p-4 gap-2 overflow-y-auto scrollbar-hide">
            {/* Logo */}
            <Link to="/" className="flex items-center gap-3 px-3 py-4 mb-2">
                <div className="w-9 h-9 nexus-gradient rounded-xl flex items-center justify-center shadow-nexus">
                    <Zap className="w-5 h-5 text-white" />
                </div>
                <span className="text-xl font-display font-black text-foreground tracking-tight">Nexus</span>
            </Link>

            {/* Nav items */}
            <div className="flex flex-col gap-1">
                {navItems.map(({ icon: Icon, label, path, badge }) => {
                    const isActive = location.pathname === path;
                    const dynamicBadge = path === '/notifications'
                        ? notifications
                        : path === '/chats'
                            ? chats
                            : badge;
                    const requiresAuth = ['/create', '/chats', '/notifications', '/profile', '/settings'].includes(path);
                    const handleClick = (e) => {
                        if (requiresAuth && !user) {
                            e.preventDefault();
                            triggerAuthModal(`Для доступа к разделу "${label}" необходимо войти в аккаунт или зарегистрироваться.`);
                        }
                    };
                    return (
                        <Link key={path} to={path} onClick={handleClick}>
                            <motion.div
                                whileHover={{ x: 2 }}
                                whileTap={{ scale: 0.98 }}
                                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all ${isActive
                                        ? 'bg-primary/10 text-primary font-semibold'
                                        : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                                    }`}
                            >
                                {isActive && (
                                    <div className="absolute left-0 w-1 h-6 nexus-gradient rounded-r-full" />
                                )}
                                <Icon className="w-5 h-5 flex-shrink-0" />
                                <span className="text-sm font-medium">{label}</span>
                                {dynamicBadge > 0 && (
                                    <Badge className="ml-auto text-[11px] h-5 min-w-5 bg-primary text-primary-foreground rounded-full px-1.5">
                                        {dynamicBadge}
                                    </Badge>
                                )}
                            </motion.div>
                        </Link>
                    );
                })}
            </div>

            {/* Admin link */}
            {(user?.role === 'admin' || user?.role === 'moderator') && (
                <Link to={user.role === 'admin' ? "/admin" : "/admin/reports"} className="mt-auto">
                    <div className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-all">
                        <Shield className="w-5 h-5" />
                        <span className="text-sm font-medium">{user.role === 'admin' ? "Администрация" : "Модерация"}</span>
                    </div>
                </Link>
            )}

            {/* User card */}
            {user && (
                <Link to="/profile" className="mt-auto">
                    <div className="flex items-center gap-3 p-3 rounded-lg bg-muted/50 hover:bg-muted transition-all">
                        <img
                            src={user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user.email}`}
                            alt="avatar"
                            className="w-8 h-8 rounded-full object-cover"
                        />
                        <div className="flex-1 min-w-0">
                            <p className="text-sm font-semibold truncate">{user.full_name || user.username}</p>
                            <p className="text-xs text-muted-foreground">Ур. {user.level || 1} · {user.xp || 0} XP</p>
                        </div>
                    </div>
                </Link>
            )}
        </aside>
    );
}
