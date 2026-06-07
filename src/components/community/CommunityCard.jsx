import { Link } from 'react-router-dom';
import { Users, Zap, TrendingUp } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { motion } from 'framer-motion';

const activityConfig = {
    low: { label: 'Спокойно', color: 'bg-muted text-muted-foreground' },
    medium: { label: 'Активно', color: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
    high: { label: 'Горячо', color: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400' },
    trending: { label: 'В тренде', color: 'bg-primary/10 text-primary' },
};

export default function CommunityCard({ community, onJoin, isJoined }) {
    const activity = activityConfig[community.activity_level] || activityConfig.low;

    return (
        <motion.div
            initial={{ opacity: 0, scale: 0.97 }}
            animate={{ opacity: 1, scale: 1 }}
            whileHover={{ y: -2 }}
            className="nexus-card overflow-hidden cursor-pointer group"
        >
            {/* Banner */}
            <Link to={`/community/${community.id}`}>
                <div className="h-20 relative overflow-hidden">
                    {community.banner_url ? (
                        <img src={community.banner_url} className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" alt="" />
                    ) : (
                        <div className="w-full h-full nexus-gradient opacity-60" />
                    )}
                    <div className="absolute inset-0 bg-gradient-to-t from-slate-950/65 via-slate-900/18 to-transparent" />
                </div>
            </Link>

            <div className="p-3 -mt-6 relative">
                <Link to={`/community/${community.id}`} className="flex items-end gap-2 mb-2">
                    <img
                        src={community.avatar_url || `https://api.dicebear.com/7.x/shapes/svg?seed=${community.name}`}
                        className="w-12 h-12 rounded-2xl border-2 border-card object-cover shadow-md flex-shrink-0"
                        alt=""
                    />
                    <div className="pb-0.5 min-w-0">
                        <h3 className="font-bold text-sm text-foreground truncate leading-tight">{community.name}</h3>
                        <div className="flex items-center gap-1 mt-0.5">
                            <Badge className={`text-[10px] px-1.5 py-0 border-0 ${activity.color}`}>
                                {activity.label}
                            </Badge>
                        </div>
                    </div>
                </Link>

                <p className="text-xs text-muted-foreground line-clamp-2 mb-3 leading-relaxed">
                    {community.description || 'Добро пожаловать в сообщество!'}
                </p>

                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 text-xs text-muted-foreground">
                        <span className="flex items-center gap-1">
                            <Users className="w-3 h-3" />
                            {(community.member_count || 0).toLocaleString()}
                        </span>
                        <span className="flex items-center gap-1">
                            <TrendingUp className="w-3 h-3" />
                            {community.post_count || 0} постов
                        </span>
                    </div>
                    <Button
                        size="sm"
                        onClick={(e) => { e.preventDefault(); onJoin?.(community); }}
                        className={`h-7 px-3 text-xs rounded-xl ${isJoined
                                ? 'border border-destructive/30 bg-white text-muted-foreground hover:bg-destructive/10 hover:border-destructive/60 hover:text-destructive dark:bg-card'
                                : 'nexus-gradient border-0 text-white shadow-nexus'
                            }`}
                    >
                        {isJoined ? 'Выйти' : 'Вступить'}
                    </Button>
                </div>
            </div>
        </motion.div>
    );
}
