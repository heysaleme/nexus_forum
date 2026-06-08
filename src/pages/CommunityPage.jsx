import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import PostCard from '@/components/feed/PostCard';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { Users, FileText, Shield, Pin, Plus, Info, ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useToast } from '@/components/ui/use-toast';

export default function CommunityPage() {
    const { id } = useParams();
    const { user } = useAuth();
    const { toast } = useToast();
    const navigate = useNavigate();
    const [community, setCommunity] = useState(null);
    const [posts, setPosts] = useState([]);
    const [members, setMembers] = useState([]);
    const [isJoined, setIsJoined] = useState(false);
    const [memberRole, setMemberRole] = useState(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => { loadCommunity(); }, [id]);

    const loadCommunity = async () => {
        setLoading(true);
        const [communityData, postsData, membersData] = await Promise.all([
            nexusApi.entities.Community.filter({ id }),
            nexusApi.entities.Post.filter({ community_id: id, status: 'published' }, '-created_date', 20),
            nexusApi.entities.CommunityMember.filter({ community_id: id }, null, 50),
        ]);
        if (communityData[0]) setCommunity(communityData[0]);
        setPosts(postsData);
        setMembers(membersData);
        if (user) {
            const myMembership = membersData.find(m => m.user_id === user.id);
            setIsJoined(!!myMembership);
            setMemberRole(myMembership?.role || null);
        }
        setLoading(false);
    };

    const handleJoin = async () => {
        if (!user) { toast({ title: 'Войдите, чтобы вступить', variant: 'destructive' }); return; }
        if (isJoined) {
            const myMember = members.find(m => m.user_id === user.id);
            if (myMember) await nexusApi.entities.CommunityMember.delete(id);
            setIsJoined(false);
            setMemberRole(null);
            toast({ title: `Вы покинули ${community.name}` });
        } else {
            await nexusApi.entities.CommunityMember.create({ user_id: user.id, community_id: id, role: 'member' });
            setIsJoined(true);
            setMemberRole('member');
            toast({ title: `Добро пожаловать в ${community.name}! 🎉` });
        }
        loadCommunity();
    };

    const handleDeleteCommunity = async () => {
        if (!window.confirm('Вы уверены, что хотите удалить это сообщество? Все связанные публикации будут также удалены!')) return;
        try {
            await nexusApi.entities.Community.delete(community.id);
            toast({ title: '🗑️ Сообщество удалено' });
            navigate('/communities');
        } catch (err) {
            toast({ title: 'Не удалось удалить сообщество', variant: 'destructive' });
        }
    };

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;
    if (!community) return <EmptyState icon={Info} title="Сообщество не найдено" />;

    const canDeleteCommunity = user && (community.owner_id === user.id || user.role === 'admin' || user.role === 'moderator');
    const pinnedPosts = posts.filter(p => p.is_pinned);
    const regularPosts = posts.filter(p => !p.is_pinned);

    return (
        <div>
            {/* Banner */}
            <div className="relative h-36 md:h-52 overflow-hidden">
                {community.banner_url ? (
                    <img src={community.banner_url} className="w-full h-full object-cover" alt="" />
                ) : (
                    <div className="w-full h-full nexus-gradient" />
                )}
                <div className="absolute inset-0 bg-gradient-to-t from-background via-background/28 to-transparent" />

                <div className="absolute top-3 left-3">
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-9 rounded-xl bg-white/75 px-3 text-foreground backdrop-blur hover:bg-white dark:bg-slate-900/70 dark:text-white dark:hover:bg-slate-900"
                        onClick={() => navigate(-1)}
                    >
                        <ArrowLeft className="w-4 h-4 mr-1.5" />
                        Назад
                    </Button>
                </div>

                {/* Join button — top right corner */}
                <div className="absolute top-3 right-3 flex gap-2">
                    {canDeleteCommunity && (
                        <Button
                            onClick={handleDeleteCommunity}
                            size="sm"
                            variant="destructive"
                            className="rounded-xl h-9 px-3 text-sm font-bold shadow bg-destructive text-white hover:bg-destructive/90"
                        >
                            Удалить
                        </Button>
                    )}
                    <Button
                        onClick={handleJoin}
                        size="sm"
                        className={`rounded-xl h-9 px-4 text-sm font-bold shadow ${isJoined
                                ? 'border border-destructive/35 bg-white/90 text-foreground hover:bg-destructive/10 hover:border-destructive/60 hover:text-destructive dark:bg-slate-950/80'
                                : 'nexus-gradient border-0 text-white shadow-nexus'
                            }`}
                    >
                        {isJoined ? 'Выйти' : 'Вступить'}
                    </Button>
                </div>
            </div>

            <div className="max-w-6xl mx-auto px-4">
                {/* Community header — avatar aligned with name */}
                <div className="flex items-center gap-3 -mt-8 mb-3 relative">
                    <img
                        src={community.avatar_url || `https://api.dicebear.com/7.x/shapes/svg?seed=${community.name}`}
                        className="w-16 h-16 rounded-xl border-3 border-background object-cover shadow-nexus flex-shrink-0"
                        style={{ borderWidth: '3px' }}
                        alt=""
                    />
                    <div className="flex-1 min-w-0 pt-6">
                        <div className="flex items-center gap-2 flex-wrap">
                            <h1 className="text-lg font-display font-black text-foreground leading-tight">{community.name}</h1>
                            {community.is_private && <Badge variant="secondary" className="text-xs">🔒 Приватное</Badge>}
                            {community.activity_level === 'trending' && <Badge className="bg-primary/10 text-primary text-xs border-0">🔥 В тренде</Badge>}
                            {(memberRole === 'moderator' || memberRole === 'owner') && (
                                <Badge className="bg-orange-100 text-orange-700 text-[10px] border-0">
                                    <Shield className="w-3 h-3 mr-1" />{memberRole === 'owner' ? 'Владелец' : 'Модератор'}
                                </Badge>
                            )}
                        </div>
                        <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
                            <span className="flex items-center gap-1"><Users className="w-3 h-3" />{community.member_count || members.length} участников</span>
                            <span className="flex items-center gap-1"><FileText className="w-3 h-3" />{community.post_count || posts.length} постов</span>
                        </div>
                    </div>
                </div>

                {community.description && (
                    <p className="text-sm text-muted-foreground mb-3 max-w-2xl">{community.description}</p>
                )}

                {/* Category + Tags row */}
                <div className="flex flex-wrap gap-1.5 mb-4">
                    {community.category && (
                        <Badge className="nexus-gradient border-0 text-white text-xs capitalize rounded">{community.category}</Badge>
                    )}
                    {community.tags?.map(tag => (
                        <Badge key={tag} variant="secondary" className="text-xs rounded bg-primary/10 text-primary border-0">#{tag}</Badge>
                    ))}
                </div>

                <div className="flex gap-6">
                    <div className="flex-1 min-w-0">
                        <Tabs defaultValue="posts">
                            <div className="flex items-center justify-between mb-4">
                                <TabsList className="bg-muted/50 rounded-xl p-1">
                                    <TabsTrigger value="posts" className="rounded-lg text-xs">Публикации</TabsTrigger>
                                    <TabsTrigger value="members" className="rounded-lg text-xs">Участники</TabsTrigger>
                                    <TabsTrigger value="rules" className="rounded-lg text-xs">Правила</TabsTrigger>
                                </TabsList>
                                {isJoined && (
                                    <Link to={`/create?community=${id}`}>
                                        <Button size="sm" className="nexus-gradient border-0 text-white rounded-xl shadow-nexus gap-1 text-xs h-8">
                                            <Plus className="w-3.5 h-3.5" />Написать
                                        </Button>
                                    </Link>
                                )}
                            </div>

                            <TabsContent value="posts">
                                {pinnedPosts.length > 0 && (
                                    <div className="mb-3">
                                        <div className="flex items-center gap-1.5 mb-2 text-xs font-semibold text-muted-foreground">
                                            <Pin className="w-3.5 h-3.5" />Закреплённые
                                        </div>
                                        <div className="nexus-feed-shell">
                                            {pinnedPosts.map(post => <PostCard key={post.id} post={post} currentUser={user} />)}
                                        </div>
                                    </div>
                                )}
                                {regularPosts.length === 0 && pinnedPosts.length === 0 ? (
                                    <EmptyState icon={FileText} title="Публикаций пока нет" description="Будь первым!" />
                                ) : (
                                    <div className="nexus-feed-shell">
                                        {regularPosts.map(post => <PostCard key={post.id} post={post} currentUser={user} />)}
                                    </div>
                                )}
                            </TabsContent>

                            <TabsContent value="members">
                                <div className="bg-card border border-border/40 overflow-hidden divide-y divide-border/30">
                                    {members.map(m => (
                                        <Link key={m.id} to={`/user/${m.user_id}`}>
                                            <div className="p-3 flex items-center gap-3 hover:bg-muted/30 transition-colors">
                                                <img src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${m.user_id}`} className="w-9 h-9 rounded-full" alt="" />
                                                <div className="flex-1">
                                                    <p className="text-sm font-semibold">{m.user_id}</p>
                                                    <p className="text-xs text-muted-foreground capitalize">{m.role}</p>
                                                </div>
                                                {(m.role === 'moderator' || m.role === 'owner') && (
                                                    <Badge className="bg-orange-100 text-orange-700 text-xs border-0">{m.role === 'owner' ? '👑 Владелец' : '🛡️ Мод'}</Badge>
                                                )}
                                            </div>
                                        </Link>
                                    ))}
                                </div>
                            </TabsContent>

                            <TabsContent value="rules">
                                {community.rules?.length > 0 ? (
                                    <div className="bg-card border border-border/40 overflow-hidden divide-y divide-border/30">
                                        {community.rules.map((rule, i) => (
                                            <div key={i} className="p-4 flex items-start gap-3">
                                                <div className="w-7 h-7 nexus-gradient rounded flex items-center justify-center flex-shrink-0 text-white text-sm font-black">{i + 1}</div>
                                                <div>
                                                    <p className="text-sm font-bold">{rule.title}</p>
                                                    {rule.description && <p className="text-xs text-muted-foreground mt-0.5">{rule.description}</p>}
                                                </div>
                                            </div>
                                        ))}
                                    </div>
                                ) : (
                                    <EmptyState icon={Shield} title="Правила не добавлены" />
                                )}
                            </TabsContent>
                        </Tabs>
                    </div>

                    {/* Right sidebar */}
                    <div className="hidden lg:block w-60 flex-shrink-0 space-y-4">
                        <div className="border border-border/40 bg-card p-4">
                            <h3 className="font-bold text-sm mb-3">Статистика</h3>
                            <div className="grid grid-cols-2 gap-2">
                                {[
                                    { label: 'Участников', value: community.member_count || members.length },
                                    { label: 'Постов', value: community.post_count || posts.length },
                                    { label: 'Активность', value: community.activity_level || 'low' },
                                    { label: 'Категория', value: community.category || '—' },
                                ].map(({ label, value }) => (
                                    <div key={label} className="bg-muted/50 p-2.5 text-center">
                                        <p className="text-sm font-black text-primary">{typeof value === 'number' ? value.toLocaleString() : value}</p>
                                        <p className="text-[10px] text-muted-foreground">{label}</p>
                                    </div>
                                ))}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
