import { Link, useLocation } from 'react-router-dom';
import { Home, Compass, PlusCircle, MessageCircle, User } from 'lucide-react';
import { motion } from 'framer-motion';

const navItems = [
    { icon: Home, label: 'Главная', path: '/' },
    { icon: Compass, label: 'Сообщества', path: '/communities' },
    { icon: PlusCircle, label: 'Создать', path: '/create', isCenter: true },
    { icon: MessageCircle, label: 'Чаты', path: '/chats' },
    { icon: User, label: 'Профиль', path: '/profile' },
];

export default function BottomNav() {
    const location = useLocation();

    return (
        <nav className="fixed bottom-0 left-0 right-0 z-50 glass border-t border-border/50 md:hidden">
            <div className="flex items-center justify-around px-2 py-2 max-w-lg mx-auto">
                {navItems.map(({ icon: Icon, label, path, isCenter }) => {
                    const isActive = location.pathname === path;
                    if (isCenter) {
                        return (
                            <Link key={path} to={path} className="flex flex-col items-center">
                                <motion.div
                                    whileTap={{ scale: 0.9 }}
                                    className="w-12 h-12 nexus-gradient rounded-2xl flex items-center justify-center shadow-nexus"
                                >
                                    <Icon className="w-6 h-6 text-white" />
                                </motion.div>
                            </Link>
                        );
                    }
                    return (
                        <Link key={path} to={path} className="flex flex-col items-center gap-1 min-w-[50px]">
                            <motion.div whileTap={{ scale: 0.9 }} className="flex flex-col items-center gap-0.5">
                                <div className={`p-1.5 rounded-xl transition-all ${isActive ? 'bg-primary/10' : ''}`}>
                                    <Icon className={`w-5 h-5 transition-colors ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                                </div>
                                <span className={`text-[10px] font-medium transition-colors ${isActive ? 'text-primary' : 'text-muted-foreground'}`}>
                                    {label}
                                </span>
                            </motion.div>
                        </Link>
                    );
                })}
            </div>
        </nav>
    );
}