import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import PostCard from '@/components/feed/PostCard';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import FollowersModal from '@/components/ui/FollowersModal';
import ReportModal from '@/components/ui/ReportModal';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { User, Star, FileText, Bookmark, Trophy, Settings, UserPlus, UserMinus, Clock, Flag } from 'lucide-react';
import { motion } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';
import { useNavigate } from 'react-router-dom';

const LEVEL_COLORS = ['bg-gray-400', 'bg-blue-400', 'bg-green-400', 'bg-yellow-400', 'bg-orange-400', 'bg-red-400', 'bg-purple-500', 'bg-pink-500', 'bg-indigo-500', 'bg-cyan-500'];

const ACHIEVEMENT_ICONS = {
    first_post: '📝',
    first_comment: '💬',
    rising_star: '⭐',
    community_builder: '🏗️',
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
    const { user: currentUser, logout } = useAuth();
    const { toast } = useToast();
    const navigate = useNavigate();

    const targetId = id || currentUser?.id;
    const isOwn = !id || id === currentUser?.id;

    const [profileUser, setProfileUser] = useState(null);
    const [posts, setPosts] = useState([]);
    const [savedPosts, setSavedPosts] = useState([]);
    const [achievements, setAchievements] = useState([]);
    const [isFollowing, setIsFollowing] = useState(false);
    const [followStatus, setFollowStatus] = useState('none');
    const [reportOpen, setReportOpen] = useState(false);
    const [loading, setLoading] = useState(true);
    // Followers modal state
    const [followersModalOpen, setFollowersModalOpen] = useState(false);
    const [followersModalTab, setFollowersModalTab] = useState('followers');

    useEffect(() => {
        if (targetId) loadProfile();
    }, [targetId, currentUser]);

    const loadProfile = async () => {
        setLoading(true);
        const [users, userPosts, userAchievements] = await Promise.all([
            nexusApi.entities.User.filter({ id: targetId }),
            nexusApi.entities.Post.filter({ author_id: targetId, status: 'published' }, '-created_date', 10),
            nexusApi.entities.Achievement.filter({ user_id: targetId }),
        ]);

        if (users[0]) setProfileUser(users[0]);
        else if (targetId === currentUser?.id) setProfileUser(currentUser);
        setPosts(userPosts || []);
        setAchievements(userAchievements || []);

        if (!isOwn && currentUser) {
            try {
                const follows = await nexusApi.entities.UserFollow.filter({ follower_id: currentUser.id, following_id: targetId });
                if (follows.length > 0) {
                    setFollowStatus(follows[0].status);
                    setIsFollowing(follows[0].status === 'accepted');
                } else {
                    setFollowStatus('none');
                    setIsFollowing(false);
                }
            } catch (err) {
                setFollowStatus('none');
                setIsFollowing(false);
            }
        }

        if (isOwn && currentUser) {
            const saved = await nexusApi.entities.SavedPost.filter({ user_id: currentUser.id });
            setSavedPosts(saved || []);
        }

        setLoading(false);
    };

    const handleFollow = async () => {
        if (!currentUser) { navigate('/login'); return; }
        if (followStatus === 'accepted' || followStatus === 'pending') {
            await nexusApi.entities.UserFollow.delete(targetId);
            const wasAccepted = followStatus === 'accepted';
            setFollowStatus('none');
            setIsFollowing(false);
            if (wasAccepted) {
                setProfileUser(prev => prev ? { ...prev, followers_count: Math.max(0, (prev.followers_count || 1) - 1) } : prev);
                toast({ title: 'Отписка оформлена' });
            } else {
                toast({ title: 'Запрос отменен' });
            }
        } else {
            await nexusApi.entities.UserFollow.create({ follower_id: currentUser.id, following_id: targetId });
            const displayUser = profileUser || currentUser;
            if (displayUser.is_private) {
                setFollowStatus('pending');
                setIsFollowing(false);
                toast({ title: '✉️ Запрос на подписку отправлен!' });
            } else {
                setFollowStatus('accepted');
                setIsFollowing(true);
                setProfileUser(prev => prev ? { ...prev, followers_count: (prev.followers_count || 0) + 1 } : prev);
                toast({ title: '✅ Вы подписались!' });
            }
        }
    };

    const handleBanUser = async () => {
        try {
            const nextBanned = !displayUser.is_banned;
            await nexusApi.entities.User.update(displayUser.id, { is_banned: nextBanned });
            setProfileUser(prev => ({ ...prev, is_banned: nextBanned }));
            toast({ title: nextBanned ? '🚫 Пользователь заблокирован' : '✅ Пользователь разблокирован' });
        } catch (err) {
            toast({ title: 'Не удалось обновить блокировку', variant: 'destructive' });
        }
    };

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;
    if (!profileUser && !currentUser) return <EmptyState icon={User} title="Пользователь не найден" />;

    const displayUser = profileUser || currentUser;
    const level = displayUser?.level || 1;
    const theme = displayUser?.profile_theme || 'default';
    const levelColor = LEVEL_COLORS[Math.min(level - 1, LEVEL_COLORS.length - 1)];

    // Isolate slash-containing classes from ternary operators in JSX to avoid esbuild parser issues
    const followBtnClass = followStatus === 'accepted'
        ? 'bg-muted text-muted-foreground hover:bg-destructive/10 hover:text-destructive'
        : followStatus === 'pending'
            ? 'bg-orange-100 text-orange-700 hover:bg-destructive/10 hover:text-destructive'
            : 'nexus-gradient border-0 text-white shadow-nexus';

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
                {isOwn && (
                    <Link to="/settings" className="absolute top-3 right-3 z-10 md:hidden">
                        <Button size="icon" variant="ghost" className="w-8 h-8 rounded-xl bg-background/60 hover:bg-background/80 backdrop-blur-sm border border-border/20 text-foreground flex items-center justify-center">
                            <Settings className="w-4 h-4" />
                        </Button>
                    </Link>
                )}
            </div>

            {/* Avatar centered on top of banner */}
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
                <div className="w-32 sm:w-36 mb-2">
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

                {/* Action buttons */}
                {isOwn ? (
                    <div className="flex gap-2 mt-2 flex-wrap justify-center">
                        <Link to="/settings">
                            <Button size="sm" variant="outline" className="rounded-xl h-8 text-xs font-bold gap-1.5">
                                <Settings className="w-3.5 h-3.5" />
                                Редактировать профиль
                            </Button>
                        </Link>
                    </div>
                ) : (
                    <div className="flex gap-2 mt-2 flex-wrap justify-center">
                        <Button
                            onClick={handleFollow}
                            size="sm"
                            className={`rounded-xl h-8 gap-1.5 text-xs font-bold ${followBtnClass}`}
                        >
                            {followStatus === 'accepted' ? (
                                <><UserMinus className="w-3.5 h-3.5" />Отписаться</>
                            ) : followStatus === 'pending' ? (
                                <><Clock className="w-3.5 h-3.5" />Запрос отправлен</>
                            ) : (
                                <><UserPlus className="w-3.5 h-3.5" />Подписаться</>
                            )}
                        </Button>
                        <Link to={`/chats?userId=${displayUser.id}`}>
                            <Button size="sm" variant="outline" className="rounded-xl h-8 text-xs font-bold gap-1.5">
                                Сообщение
                            </Button>
                        </Link>
                        {currentUser && (
                            <Button
                                onClick={() => setReportOpen(true)}
                                size="sm"
                                variant="outline"
                                className="rounded-xl h-8 text-xs font-bold gap-1.5 text-orange-600 hover:text-orange-700 hover:bg-orange-50 border-orange-200"
                            >
                                <Flag className="w-3.5 h-3.5" />
                                Пожаловаться
                            </Button>
                        )}
                        {currentUser && (currentUser.role === 'admin' || currentUser.role === 'moderator') && (
                            <Button
                                onClick={handleBanUser}
                                size="sm"
                                variant={displayUser.is_banned ? 'outline' : 'destructive'}
                                className="rounded-xl h-8 text-xs font-bold"
                            >
                                {displayUser.is_banned ? 'Разблокировать' : 'Заблокировать'}
                            </Button>
                        )}
                    </div>
                )}
            </div>

            {/* Stats — three equal columns with clickable followers/following */}
            <div className="flex border-b border-border/30">
                <div className="flex-1 flex flex-col items-center py-3 border-r border-border/30">
                    <span className="text-sm font-black text-foreground">{(displayUser.is_private && !isOwn && followStatus !== 'accepted') ? '—' : (displayUser?.xp || 0)}</span>
                    <span className="text-[10px] text-muted-foreground flex items-center gap-0.5"><Star className="w-2.5 h-2.5 text-yellow-500" />XP</span>
                </div>
                <button
                    onClick={() => {
                        if (!displayUser.is_private || isOwn || followStatus === 'accepted') {
                            setFollowersModalTab('followers');
                            setFollowersModalOpen(true);
                        }
                    }}
                    className="flex-1 flex flex-col items-center py-3 border-r border-border/30 hover:bg-muted/30 transition-colors"
                    disabled={displayUser.is_private && !isOwn && followStatus !== 'accepted'}
                >
                    <span className="text-sm font-black text-foreground">{(displayUser.is_private && !isOwn && followStatus !== 'accepted') ? '—' : (displayUser?.followers_count || 0)}</span>
                    <span className="text-[10px] text-muted-foreground">подписчиков</span>
                </button>
                <button
                    onClick={() => {
                        if (!displayUser.is_private || isOwn || followStatus === 'accepted') {
                            setFollowersModalTab('following');
                            setFollowersModalOpen(true);
                        }
                    }}
                    className="flex-1 flex flex-col items-center py-3 hover:bg-muted/30 transition-colors"
                    disabled={displayUser.is_private && !isOwn && followStatus !== 'accepted'}
                >
                    <span className="text-sm font-black text-foreground">{(displayUser.is_private && !isOwn && followStatus !== 'accepted') ? '—' : (displayUser?.following_count || 0)}</span>
                    <span className="text-[10px] text-muted-foreground">подписок</span>
                </button>
            </div>

            {/* Followers modal */}
            <FollowersModal
                open={followersModalOpen}
                onClose={() => setFollowersModalOpen(false)}
                userId={displayUser?.id}
                defaultTab={followersModalTab}
            />

            {/* Tabs or Private Lock */}
            {displayUser.is_private && !isOwn && followStatus !== 'accepted' ? (
                <div className="flex flex-col items-center justify-center py-20 px-4 text-center my-6">
                    <div className="w-16 h-16 bg-muted rounded-full flex items-center justify-center mb-4 text-muted-foreground text-2xl border border-border">
                        🔒
                    </div>
                    <h3 className="text-base font-bold text-foreground mb-1">Это приватный аккаунт</h3>
                    <p className="text-xs text-muted-foreground max-w-xs leading-relaxed">
                        Подпишитесь на этого пользователя, чтобы видеть его публикации и достижения.
                    </p>
                </div>
            ) : (
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
                                                <div className={['flex items-center gap-3 p-3 hover:bg-muted/30 transition-colors', i !== 0 ? 'border-t border-border/30' : ''].join(' ')}>
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
                                    <p>Пост: +20 XP.</p>
                                    <p>Комментарий: +8 XP.</p>
                                    <p>Создание сообщества: +30 XP.</p>
                                    <p>Подписка на пользователя: +4 XP.</p>
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
                                            <Badge className={`text-[9px] border-0 ${a.tier === 'platinum' ? 'bg-cyan-100 text-cyan-700' : a.tier === 'gold' ? 'bg-yellow-100 text-yellow-700' : a.tier === 'silver' ? 'bg-gray-100 text-gray-600' : 'bg-orange-100 text-orange-700'}`}>{a.tier}</Badge>
                                        </motion.div>
                                    ))}
                                </div>
                            )}
                        </TabsContent>
                    </Tabs>
                </div>
            )}

            {reportOpen && (
                <ReportModal
                    open={reportOpen}
                    onClose={() => setReportOpen(false)}
                    targetId={displayUser.id}
                    targetType="user"
                    currentUser={currentUser}
                />
            )}
        </div>
    );
}
