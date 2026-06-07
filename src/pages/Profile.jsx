import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { base44 } from '@/api/base44Client';
import { useAuth } from '@/lib/AuthContext';
import PostCard from '@/components/feed/PostCard';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { User, Star, FileText, Bookmark, Trophy, Settings, UserPlus, UserMinus } from 'lucide-react';
import { motion } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';
import { useNavigate } from 'react-router-dom';

const LEVEL_COLORS = ['bg-gray-400', 'bg-blue-400', 'bg-green-400', 'bg-yellow-400', 'bg-orange-400', 'bg-red-400', 'bg-purple-500', 'bg-pink-500', 'bg-indigo-500', 'bg-cyan-500'];

const ACHIEVEMENT_ICONS = {
    first_post: '📝',
    first_comment: '💬',
    rising_star: '⭐',
    community_builder: '🏗️',
    wiki_master: '📚',
    social_butterfly: '🦋',
    veteran: '🎖️',
    top_contributor: '🏆',
    trendsetter: '🔥',
    supporter: '💜',
};

const THEME_GRADIENTS = {
    default: 'from-primary/20 to-accent/20',
    purple: 'from-purple-400/30 to-indigo-500/30',
    ocean: 'from-blue-400/30 to-cyan-400/30',
    sunset: 'from-orange-400/30 to-pink-500/30',
    forest: 'from-green-400/30 to-emerald-500/30',
    dark: 'from-gray-700/50 to-gray-900/50',
};

export default function Profile() {
    const { id } = useParams();
    const { user: currentUser } = useAuth();
    const { toast } = useToast();
    const navigate = useNavigate();

    const targetId = id || currentUser?.id;
    const isOwn = !id || id === currentUser?.id;

    const [profileUser, setProfileUser] = useState(null);
    const [posts, setPosts] = useState([]);
    const [savedPosts, setSavedPosts] = useState([]);
    const [achievements, setAchievements] = useState([]);
    const [isFollowing, setIsFollowing] = useState(false);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (targetId) loadProfile();
    }, [targetId, currentUser]);

    const loadProfile = async () => {
        setLoading(true);
        const [users, userPosts, userAchievements] = await Promise.all([
            base44.entities.User.filter({ id: targetId }),
            base44.entities.Post.filter({ author_id: targetId, status: 'published' }, '-created_date', 10),
            base44.entities.Achievement.filter({ user_id: targetId }),
        ]);

        if (users[0]) setProfileUser(users[0]);
        else if (targetId === currentUser?.id) setProfileUser(currentUser);
        setPosts(userPosts);
        setAchievements(userAchievements);

        if (!isOwn && currentUser) {
            const follows = await base44.entities.UserFollow.filter({ follower_id: currentUser.id, following_id: targetId });
            setIsFollowing(follows.length > 0);
        }

        if (isOwn && currentUser) {
            const saved = await base44.entities.SavedPost.filter({ user_id: currentUser.id });
            setSavedPosts(saved);
        }

        setLoading(false);
    };

    const handleFollow = async () => {
        if (!currentUser) { navigate('/login'); return; }
        if (isFollowing) {
            const follows = await base44.entities.UserFollow.filter({ follower_id: currentUser.id, following_id: targetId });
            if (follows[0]) await base44.entities.UserFollow.delete(follows[0].id);
            setIsFollowing(false);
            toast({ title: 'Вы отписались' });
        } else {
            await base44.entities.UserFollow.create({ follower_id: currentUser.id, following_id: targetId });
            setIsFollowing(true);
            toast({ title: '✅ Вы подписались!' });
        }
        loadProfile();
    };

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;
    if (!profileUser && !currentUser) return <EmptyState icon={User} title="Пользователь не найден" />;

    const displayUser = profileUser || currentUser;
    const level = displayUser?.level || 1;
    const theme = displayUser?.profile_theme || 'default';
    const levelColor = LEVEL_COLORS[Math.min(level - 1, LEVEL_COLORS.length - 1)];

    return (
        <div>
            {/* Banner */}
            <div className={`relative h-32 md:h-44 overflow-hidden bg-gradient-to-br ${THEME_GRADIENTS[theme]}`}>
                {displayUser?.banner_url ? (
                    <img src={displayUser.banner_url} className="w-full h-full object-cover" alt="" />
                ) : (
                    <div className="w-full h-full nexus-gradient opacity-30" />
                )}
                <div className="absolute inset-0 bg-gradient-to-t from-background/40 to-transparent" />
            </div>

            {/* Avatar — centered, round, on top of banner */}
            <div className="flex flex-col items-center -mt-14 px-4 pb-3 border-b border-border/30">
                <div className="relative mb-2">
                    <img
                        src={displayUser?.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${displayUser?.email}`}
                        className="w-24 h-24 rounded-full border-4 border-background object-cover shadow-nexus"
                        alt=""
                    />
                </div>

                {/* Username */}
                <h1 className="text-lg font-display font-black text-foreground text-center">
                    {displayUser?.full_name || displayUser?.username || 'Пользователь'}
                </h1>
                {/* {displayUser?.title && (
                    <p className="text-xs font-semibold text-primary mb-1">{displayUser.title}</p>
                )} */}
                <div className="w-[7.5rem] sm:w-[8.5rem] mb-2">
                    <div className="flex items-center justify-between mb-1">
                        <span className="text-[10px] font-bold text-foreground">Уровень {level}</span>
                        <span className="text-[10px] text-muted-foreground">{displayUser?.xp || 0} XP</span>
                    </div>
                    <div className="h-2 bg-muted rounded-full overflow-hidden">
                        <motion.div
                            initial={{ width: 0 }}
                            animate={{ width: `${Math.min(100, ((displayUser?.xp || 0) % 100))}%` }}
                            transition={{ duration: 0.8, ease: 'easeOut' }}
                            className={`h-full ${levelColor} rounded-full`}
                        />
                    </div>
                </div>
                {displayUser?.bio && (
                    <p className="text-xs text-muted-foreground text-center max-w-xs mb-2">{displayUser.bio}</p>
                )}

                {/* Action button — settings only on mobile for own profile */}
                {isOwn ? (
                    <Link to="/settings" className="md:hidden">
                        <Button variant="outline" size="sm" className="rounded gap-1.5 text-xs h-8 mt-1">
                            <Settings className="w-3.5 h-3.5" />
                            Настройки
                        </Button>
                    </Link>
                ) : (
                    <Button
                        onClick={handleFollow}
                        size="sm"
                        className={`rounded-xl h-8 gap-1.5 text-xs font-bold mt-1 ${isFollowing ? 'bg-muted text-muted-foreground hover:bg-destructive/10 hover:text-destructive' : 'nexus-gradient border-0 text-white shadow-nexus'
                            }`}
                    >
                        {isFollowing ? <><UserMinus className="w-3.5 h-3.5" />Отписаться</> : <><UserPlus className="w-3.5 h-3.5" />Подписаться</>}
                    </Button>
                )}
            </div>

            {/* Stats — three equal columns separated by thin dividers */}
            <div className="flex border-b border-border/30">
                <div className="flex-1 flex flex-col items-center py-3 border-r border-border/30">
                    <span className="text-sm font-black text-foreground">{displayUser?.karma || 0}</span>
                    <span className="text-[10px] text-muted-foreground flex items-center gap-0.5"><Star className="w-2.5 h-2.5 text-yellow-500" />karma</span>
                </div>
                <div className="flex-1 flex flex-col items-center py-3 border-r border-border/30">
                    <span className="text-sm font-black text-foreground">{displayUser?.followers_count || 0}</span>
                    <span className="text-[10px] text-muted-foreground">подписчиков</span>
                </div>
                <div className="flex-1 flex flex-col items-center py-3">
                    <span className="text-sm font-black text-foreground">{displayUser?.following_count || 0}</span>
                    <span className="text-[10px] text-muted-foreground">подписок</span>
                </div>
            </div>

            {/* Tabs */}
            <div className="px-4 pt-3">
                <Tabs defaultValue="posts">
                    <TabsList className="bg-muted/50 rounded-xl p-1 mb-3 w-full">
                        <TabsTrigger value="posts" className="rounded-lg text-xs gap-1.5 flex-1">
                            <FileText className="w-3.5 h-3.5" />Посты ({posts.length})
                        </TabsTrigger>
                        {isOwn && (
                            <TabsTrigger value="saved" className="rounded-lg text-xs gap-1.5 flex-1">
                                <Bookmark className="w-3.5 h-3.5" />Сохранённые
                            </TabsTrigger>
                        )}
                        <TabsTrigger value="achievements" className="rounded-lg text-xs gap-1.5 flex-1">
                            <Trophy className="w-3.5 h-3.5" />Достижения
                        </TabsTrigger>
                    </TabsList>

                    <TabsContent value="posts">
                        {posts.length === 0 ? (
                            <EmptyState icon={FileText} title="Публикаций пока нет" />
                        ) : (
                            <div className="nexus-feed-shell">
                                {posts.map(post => <PostCard key={post.id} post={post} currentUser={currentUser} />)}
                            </div>
                        )}
                    </TabsContent>

                    {isOwn && (
                        <TabsContent value="saved">
                            {savedPosts.length === 0 ? (
                                <EmptyState icon={Bookmark} title="Нет сохранённых постов" />
                            ) : (
                                <div className="space-y-px bg-card rounded-2xl overflow-hidden border border-border/40">
                                    {savedPosts.map((s, i) => (
                                        <Link key={s.id} to={`/post/${s.post_id}`}>
                                            <div className={`flex items-center gap-3 p-3 hover:bg-muted/30 transition-colors ${i > 0 ? 'border-t border-border/30' : ''}`}>
                                                <Bookmark className="w-4 h-4 text-primary flex-shrink-0" />
                                                <div>
                                                    <p className="text-sm font-semibold line-clamp-1">{s.post_title}</p>
                                                    <p className="text-xs text-muted-foreground">{s.post_community}</p>
                                                </div>
                                            </div>
                                        </Link>
                                    ))}
                                </div>
                            )}
                        </TabsContent>
                    )}

                    <TabsContent value="achievements">
                        <div className="nexus-card p-4 mb-3">
                            <p className="text-sm font-bold mb-2">Как получать XP и достижения</p>
                            <div className="space-y-1 text-xs text-muted-foreground">
                                <p>Пост: `+20 XP` и `+5 karma`.</p>
                                <p>Комментарий: `+8 XP` и `+2 karma`.</p>
                                <p>Создание сообщества: `+30 XP` и `+10 karma`.</p>
                                <p>Подписка на пользователя: `+4 XP` и `+1 karma`.</p>
                            </div>
                        </div>
                        {achievements.length === 0 ? (
                            <EmptyState icon={Trophy} title="Достижений пока нет" description="Публикуй посты и комментируй для получения наград!" />
                        ) : (
                            <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                                {achievements.map(a => (
                                    <motion.div
                                        key={a.id}
                                        initial={{ opacity: 0, scale: 0.9 }}
                                        animate={{ opacity: 1, scale: 1 }}
                                        className="nexus-card p-4 text-center"
                                    >
                                        <div className="text-3xl mb-2">{ACHIEVEMENT_ICONS[a.achievement_name] || '🏅'}</div>
                                        <p className="text-xs font-bold text-foreground mb-0.5">{a.achievement_description || a.achievement_name}</p>
                                        <Badge className={`text-[9px] border-0 ${a.tier === 'platinum' ? 'bg-cyan-100 text-cyan-700' :
                                            a.tier === 'gold' ? 'bg-yellow-100 text-yellow-700' :
                                                a.tier === 'silver' ? 'bg-gray-100 text-gray-600' :
                                                    'bg-orange-100 text-orange-700'
                                            }`}>{a.tier}</Badge>
                                    </motion.div>
                                ))}
                            </div>
                        )}
                    </TabsContent>
                </Tabs>
            </div>
        </div>
    );
}
