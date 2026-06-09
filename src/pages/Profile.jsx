import { useState, useEffect } from 'react';
import { useParams, Link, useSearchParams } from 'react-router-dom';
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
import { User, FileText, Bookmark, Trophy, Settings, UserPlus, UserMinus, Clock, Flag, EyeOff, ArrowUpCircle } from 'lucide-react';
import { motion } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';
import { useNavigate } from 'react-router-dom';
import { isOwnProfile } from '@/lib/profileLink';

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
    const [searchParams, setSearchParams] = useSearchParams();
    const activeTab = searchParams.get('tab') || 'posts';
    const { user: currentUser, logout } = useAuth();
    const { toast } = useToast();
    const navigate = useNavigate();

    const targetId = id || currentUser?.id;
    const isOwn = !id || (currentUser && isOwnProfile(id, currentUser.id));

    const [profileUser, setProfileUser] = useState(null);
    const [posts, setPosts] = useState([]);
    const [drafts, setDrafts] = useState([]);
    const [scheduled, setScheduled] = useState([]);
    const [savedPosts, setSavedPosts] = useState([]);
    const [achievements, setAchievements] = useState([]);
    const [profileStats, setProfileStats] = useState(null);
    const [isFollowing, setIsFollowing] = useState(false);
    const [followStatus, setFollowStatus] = useState('none');
    const [reportOpen, setReportOpen] = useState(false);
    const [loading, setLoading] = useState(true);
    // Followers modal state
    const [followersModalOpen, setFollowersModalOpen] = useState(false);
    const [followersModalTab, setFollowersModalTab] = useState('followers');

    useEffect(() => {
        if (id && currentUser && isOwnProfile(id, currentUser.id)) {
            navigate(`/profile${searchParams.toString() ? `?${searchParams.toString()}` : ''}`, { replace: true });
        }
    }, [id, currentUser, navigate, searchParams]);

    useEffect(() => {
        if (targetId) loadProfile();
    }, [targetId, currentUser, isOwn]);

    const loadProfile = async () => {
        setLoading(true);
        try {
            const ownProfile = isOwn;
            const authorId = Number(targetId);
            const [users, userPosts, userDrafts, userScheduled, userAchievements, stats] = await Promise.all([
                nexusApi.entities.User.filter({ id: authorId }),
                nexusApi.entities.Post.filter({ author_id: authorId, status: 'published' }, '-created_date', 10),
                ownProfile ? nexusApi.entities.Post.filter({ author_id: authorId, status: 'draft' }, '-created_date', 20) : Promise.resolve([]),
                ownProfile ? nexusApi.entities.Post.filter({ author_id: authorId, status: 'scheduled' }, '-created_date', 20) : Promise.resolve([]),
                nexusApi.entities.Achievement.filter({ user_id: authorId }).catch(() => []),
                nexusApi.entities.User.getStats(authorId).catch(() => null),
            ]);

            if (users[0]) setProfileUser(users[0]);
            else if (targetId === currentUser?.id) setProfileUser(currentUser);
            setPosts((userPosts || []).filter((p) => p?.id && p.status !== 'removed'));
            setDrafts((userDrafts || []).filter((p) => p?.id));
            setScheduled((userScheduled || []).filter((p) => p?.id));
            setAchievements(userAchievements || []);
            setProfileStats(stats);

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
                } catch {
                    setFollowStatus('none');
                    setIsFollowing(false);
                }
            }

            if (isOwn && currentUser) {
                const saved = await nexusApi.entities.SavedPost.filter({ user_id: currentUser.id });
                setSavedPosts(saved || []);
            }
        } catch (err) {
            console.error('Profile load failed:', err);
            toast({ title: 'Не удалось загрузить профиль', variant: 'destructive' });
        } finally {
            setLoading(false);
        }
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

    const canModerate = currentUser && (currentUser.role === 'admin' || currentUser.role === 'moderator') && !isOwn;

    const handleBanUser = async () => {
        const target = profileUser || currentUser;
        if (!target?.id) return;
        try {
            const nextBanned = !target.is_banned;
            if (nextBanned) {
                await nexusApi.entities.Moderation.banUser(target.id);
            } else {
                await nexusApi.entities.Moderation.unbanUser(target.id);
            }
            setProfileUser((prev) => (prev ? { ...prev, is_banned: nextBanned } : prev));
            toast({ title: nextBanned ? '🚫 Пользователь заблокирован' : '✅ Пользователь разблокирован' });
        } catch {
            toast({ title: 'Не удалось обновить блокировку', variant: 'destructive' });
        }
    };

    const handleShadowBanUser = async () => {
        const target = profileUser;
        if (!target?.id) return;
        try {
            const nextShadow = !target.is_shadow_banned;
            if (nextShadow) {
                await nexusApi.entities.Moderation.shadowBanUser(target.id);
            } else {
                await nexusApi.entities.Moderation.unshadowBanUser(target.id);
            }
            setProfileUser((prev) => (prev ? { ...prev, is_shadow_banned: nextShadow } : prev));
            toast({ title: nextShadow ? '🌑 Теневой бан применён' : '👁️ Теневой бан снят' });
        } catch {
            toast({ title: 'Не удалось обновить теневой бан', variant: 'destructive' });
        }
    };

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;
    if (!profileUser && !currentUser) return <EmptyState icon={User} title="Пользователь не найден" />;

    const displayUser = profileUser || currentUser;
    const theme = displayUser?.profile_theme || 'default';
    const postKarma = displayUser?.post_karma ?? 0;
    const commentKarma = displayUser?.comment_karma ?? 0;
    const totalKarma = displayUser?.total_karma ?? (postKarma + commentKarma);
    const karmaHidden = displayUser.is_private && !isOwn && followStatus !== 'accepted';

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
                {canModerate && (
                    <div className="flex flex-wrap gap-1.5 justify-center mb-1">
                        {displayUser?.is_banned && <Badge className="bg-destructive/10 text-destructive text-[9px] border-0">Заблокирован</Badge>}
                        {displayUser?.is_shadow_banned && <Badge className="bg-orange-100 text-orange-700 text-[9px] border-0">Теневой бан</Badge>}
                        {!displayUser?.is_banned && !displayUser?.is_shadow_banned && (
                            <Badge className="bg-green-100 text-green-700 text-[9px] border-0">Активен</Badge>
                        )}
                    </div>
                )}
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
                        {canModerate && (
                            <>
                                <Button
                                    onClick={handleBanUser}
                                    size="sm"
                                    variant={displayUser.is_banned ? 'outline' : 'destructive'}
                                    className="rounded-xl h-8 text-xs font-bold"
                                >
                                    {displayUser.is_banned ? 'Разблокировать' : 'Заблокировать'}
                                </Button>
                                <Button
                                    onClick={handleShadowBanUser}
                                    size="sm"
                                    variant={displayUser.is_shadow_banned ? 'outline' : 'secondary'}
                                    className="rounded-xl h-8 text-xs font-bold gap-1.5"
                                >
                                    <EyeOff className="w-3.5 h-3.5" />
                                    {displayUser.is_shadow_banned ? 'Снять теневой бан' : 'Теневой бан'}
                                </Button>
                            </>
                        )}
                    </div>
                )}
            </div>

            {/* Stats — three equal columns with clickable followers/following */}
            <div className="flex border-b border-border/30">
                <div className="flex-1 flex flex-col items-center py-3 border-r border-border/30">
                    <span className="text-sm font-black text-foreground">{karmaHidden ? '—' : totalKarma}</span>
                    <span className="text-[10px] text-muted-foreground flex items-center gap-0.5"><ArrowUpCircle className="w-2.5 h-2.5 text-primary" />карма</span>
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

            {profileStats && (
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 px-4 py-3 border-b border-border/30 text-center">
                    <div><span className="text-sm font-black block">{profileStats.posts_count ?? 0}</span><span className="text-[10px] text-muted-foreground">постов</span></div>
                    <div><span className="text-sm font-black block">{profileStats.comments_count ?? 0}</span><span className="text-[10px] text-muted-foreground">комментариев</span></div>
                    <div><span className="text-sm font-black block">{profileStats.communities_count ?? 0}</span><span className="text-[10px] text-muted-foreground">сообществ</span></div>
                    <div><span className="text-sm font-black block">{isOwn ? (profileStats.saved_count ?? 0) : (profileStats.achievements_count ?? achievements.length)}</span><span className="text-[10px] text-muted-foreground">{isOwn ? 'сохранено' : 'достижений'}</span></div>
                </div>
            )}

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
                    <Tabs value={activeTab} onValueChange={(tab) => setSearchParams(tab === 'posts' ? {} : { tab })}>
                        <TabsList className="bg-muted/50 rounded-xl p-1 mb-3 w-full">
                            {[
                                { value: 'posts', icon: FileText, label: `Посты (${profileStats?.posts_count ?? posts.length})` },
                                ...(isOwn ? [
                                    { value: 'drafts', icon: FileText, label: `Черновики (${drafts.length})` },
                                    { value: 'scheduled', icon: Clock, label: `Отложенные (${scheduled.length})` },
                                    { value: 'saved', icon: Bookmark, label: `Сохранённые (${profileStats?.saved_count ?? savedPosts.length})` },
                                ] : []),
                                { value: 'achievements', icon: Trophy, label: `Достижения (${profileStats?.achievements_count ?? achievements.length})` },
                            ].map(({ value, icon: Icon, label }) => (
                                <TabsTrigger key={value} value={value} className="rounded-lg text-xs gap-1.5 flex-1 px-2">
                                    <Icon className="w-3.5 h-3.5 shrink-0" />
                                    <span className={`truncate ${activeTab === value ? 'inline' : 'hidden'} md:inline`}>{label}</span>
                                </TabsTrigger>
                            ))}
                        </TabsList>

                        {isOwn && (
                            <TabsContent value="drafts">
                                {drafts.length === 0 ? (
                                    <EmptyState icon={FileText} title="Черновиков нет" description="Сохраните пост как черновик при создании" />
                                ) : (
                                    <div className="nexus-feed-shell">
                                        {drafts.map((post) => (
                                            <div key={post.id} className="nexus-card p-4 mb-3">
                                                <p className="text-sm font-bold mb-1">{post.title}</p>
                                                <p className="text-xs text-muted-foreground line-clamp-2 mb-3">{post.content}</p>
                                                <div className="flex gap-2">
                                                    <Link to={`/create?draft=${post.id}`}>
                                                        <Button size="sm" variant="outline" className="rounded-xl h-8 text-xs">Редактировать</Button>
                                                    </Link>
                                                    <Button
                                                        size="sm"
                                                        className="rounded-xl h-8 text-xs nexus-gradient border-0 text-white"
                                                        onClick={async () => {
                                                            await nexusApi.entities.Post.update(post.id, { status: 'published' });
                                                            toast({ title: 'Опубликовано' });
                                                            loadProfile();
                                                        }}
                                                    >
                                                        Опубликовать
                                                    </Button>
                                                    <Button
                                                        size="sm"
                                                        variant="destructive"
                                                        className="rounded-xl h-8 text-xs"
                                                        onClick={async () => {
                                                            await nexusApi.entities.Post.delete(post.id);
                                                            toast({ title: 'Черновик удалён' });
                                                            loadProfile();
                                                        }}
                                                    >
                                                        Удалить
                                                    </Button>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </TabsContent>
                        )}

                        {isOwn && (
                            <TabsContent value="scheduled">
                                {scheduled.length === 0 ? (
                                    <EmptyState icon={Clock} title="Нет отложенных публикаций" description="Задайте дату публикации при создании поста" />
                                ) : (
                                    <div className="nexus-feed-shell">
                                        {scheduled.map((post) => (
                                            <div key={post.id} className="nexus-card p-4 mb-3">
                                                <p className="text-sm font-bold mb-1">{post.title}</p>
                                                <p className="text-xs text-muted-foreground mb-2">
                                                    Публикация: {post.publish_at ? new Date(post.publish_at).toLocaleString() : '—'}
                                                </p>
                                                <div className="flex gap-2">
                                                    <Link to={`/create?draft=${post.id}`}>
                                                        <Button size="sm" variant="outline" className="rounded-xl h-8 text-xs">Изменить</Button>
                                                    </Link>
                                                    <Button
                                                        size="sm"
                                                        variant="destructive"
                                                        className="rounded-xl h-8 text-xs"
                                                        onClick={async () => {
                                                            await nexusApi.entities.Post.delete(post.id);
                                                            toast({ title: 'Отложенная публикация отменена' });
                                                            loadProfile();
                                                        }}
                                                    >
                                                        Отменить
                                                    </Button>
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                )}
                            </TabsContent>
                        )}

                        <TabsContent value="posts">
                            {posts.length === 0 ? (
                                <EmptyState icon={FileText} title="Публикаций пока нет" />
                            ) : (
                                <div className="nexus-feed-shell">
                                    {posts.map(post => (
                                        <PostCard
                                            key={post.id}
                                            post={post}
                                            currentUser={currentUser}
                                            onDeleteSuccess={(postId) => setPosts((prev) => prev.filter((p) => p.id !== postId))}
                                        />
                                    ))}
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
