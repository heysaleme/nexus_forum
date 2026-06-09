import { motion } from 'framer-motion';
import { Flame, Clock, TrendingUp, Users, Zap, Building2 } from 'lucide-react';

const sorts = [
    { id: 'hot', label: 'Горячее', icon: Flame },
    { id: 'new', label: 'Новое', icon: Clock },
    { id: 'top', label: 'Лучшее', icon: TrendingUp },
    { id: 'following-users', label: 'Following Users', icon: Users },
    { id: 'following-communities', label: 'Following Communities', icon: Building2 },
    { id: 'trending', label: 'Тренды', icon: Zap },
];

export default function SortBar({ active, onChange }) {
    return (
        <div className="flex gap-2 overflow-x-auto scrollbar-hide px-1 py-1">
            {sorts.map(({ id, label, icon: Icon }) => (
                <motion.button
                    key={id}
                    whileTap={{ scale: 0.95 }}
                    onClick={() => onChange(id)}
                    className={`flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold whitespace-nowrap transition-all flex-shrink-0 ${active === id
                            ? 'nexus-gradient text-white shadow-nexus'
                            : 'bg-muted/60 text-muted-foreground hover:bg-muted hover:text-foreground'
                        }`}
                >
                    <Icon className="w-3.5 h-3.5" />
                    {label}
                </motion.button>
            ))}
        </div>
    );
}