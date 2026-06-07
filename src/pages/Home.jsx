import { useState, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import PostCard from '@/components/feed/PostCard';
import SortBar from '@/components/feed/SortBar';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { FileText, TrendingUp, Users, Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { motion } from 'framer-motion';

const CATEGORIES = [
    { id: 'all', label: 'Все интересы' },
    { id: 'anime', label: 'Аниме' },
    { id: 'gaming', label: 'Игры' },
    { id: 'fandoms', label: 'Фандомы' },
    { id: 'roleplay', label: 'РП' },
    { id: 'art', label: 'Искусство' },
    { id: 'music', label: 'Музыка' },
    { id: 'books', label: 'Книги' },
    { id: 'movies', label: 'Кино' },
    { id: 'technology', label: 'Технологии' },
    { id: 'lifestyle', label: 'Лайфстайл' },
];

export default function Home() {
    const { user } = useAuth();
    const [posts, setPosts] = useState([]);
    const [loading, setLoading] = useState(true);
    const [sort, setSort] = useState('hot');
    const [communities, setCommunities] = useState([]);
    const [allCommunities, setAllCommunities] = useState([]);
    const [activeCategory, setActiveCategory] = useState('all');

    useEffect(() => {
        loadData();
    }, [sort, user, activeCategory]);

    const loadData = async () => {
        setLoading(true);

        const [allPosts, communitiesData, popularComm] = await Promise.all([
            nexusApi.entities.Post.filter({ status: 'published' }, '-created_date', 50),
            nexusApi.entities.Community.list('-name', 100).catch(() => []),
            nexusApi.entities.Community.list('-member_count', 5).catch(() => []),
        ]);

        setCommunities(popularComm);
        setAllCommunities(communitiesData);

        let filtered = allPosts;

        // Filter by category
        if (activeCategory !== 'all') {
            filtered = filtered.filter(p => {
                const comm = communitiesData.find(c => c.id === p.community_id);
                return comm?.category === activeCategory;
            });
        }

        // Filter by followed communities
        if (sort === 'following' && user) {
            const memberships = await nexusApi.entities.CommunityMember.filter({ user_id: user.id });
            const joinedIds = new Set(memberships.map(m => m.community_id));
            filtered = filtered.filter(p => joinedIds.has(p.community_id));
        } else if (sort === 'following' && !user) {
            filtered = [];
        }

        // Sort
        if (sort === 'top') {
            filtered = [...filtered].sort((a, b) => (b.score || 0) - (a.score || 0));
        } else if (sort === 'trending') {
            filtered = [...filtered].sort((a, b) => ((b.views || 0) + (b.score || 0) * 2) - ((a.views || 0) + (a.score || 0) * 2));
        } else if (sort === 'new' || sort === 'following') {
            filtered = [...filtered].sort((a, b) => new Date(b.created_date) - new Date(a.created_date));
        }
        // 'hot' stays as default -created_date

        setPosts(filtered.slice(0, 20));
        setLoading(false);
    };

    return (
        <div className="max-w-7xl mx-auto px-4 py-4">
            <div className="flex gap-6">
                {/* Main feed */}
                <div className="flex-1 min-w-0">
                    {/* Hero banner (for guests) */}
                    {!user && (
                        <motion.div
                            initial={{ opacity: 0, y: -10 }}
                            animate={{ opacity: 1, y: 0 }}
                            className="nexus-gradient rounded-3xl p-6 mb-5 text-white relative overflow-hidden"
                        >
                            <div className="absolute top-0 right-0 w-48 h-48 bg-white/10 rounded-full -translate-y-12 translate-x-12" />
                            <div className="absolute bottom-0 left-0 w-32 h-32 bg-white/10 rounded-full translate-y-8 -translate-x-8" />
                            <div className="relative">
                                <div className="flex items-center gap-2 mb-2">
                                    <Sparkles className="w-5 h-5" />
                                    <span className="text-sm font-semibold opacity-80">Добро пожаловать в</span>
                                </div>
                                <h1 className="text-3xl font-display font-black mb-2">Nexus</h1>
                                <p className="text-sm opacity-80 mb-4 max-w-xs">
                                    Платформа для фандомов, ролевых игр и творческих сообществ. Найди своих!
                                </p>
                                <div className="flex gap-2">
                                    <Link to="/register">
                                        <Button className="bg-white text-primary hover:bg-white/90 rounded-xl font-bold text-sm h-9 px-4">
                                            Присоединиться
                                        </Button>
                                    </Link>
                                    <Link to="/communities">
                                        <Button variant="ghost" className="text-white hover:bg-white/20 rounded-xl text-sm h-9 px-4">
                                            Обзор
                                        </Button>
                                    </Link>
                                </div>
                            </div>
                        </motion.div>
                    )}

                    {/* Categories */}
                    <div className="flex gap-2 overflow-x-auto scrollbar-hide mb-4 pb-1">
                        {CATEGORIES.map(({ id, label }) => (
                            <motion.button
                                key={id}
                                whileTap={{ scale: 0.95 }}
                                onClick={() => setActiveCategory(id)}
                                className={`px-3.5 py-1.5 rounded-xl text-sm font-semibold whitespace-nowrap flex-shrink-0 transition-all ${activeCategory === id
                                    ? 'nexus-gradient text-white shadow-nexus'
                                    : 'bg-muted/60 text-muted-foreground hover:bg-muted'
                                    }`}
                            >
                                {label}
                            </motion.button>
                        ))}
                    </div>

                    {/* Sort bar */}
                    <div className="mb-3">
                        <SortBar active={sort} onChange={setSort} />
                    </div>

                    {/* Posts */}
                    {loading ? (
                        <LoadingSpinner size="lg" className="py-20" />
                    ) : posts.length === 0 ? (
                        <EmptyState
                            icon={FileText}
                            title={sort === 'following' ? 'Нет постов из подписок' : 'Лента пуста'}
                            description={sort === 'following' ? 'Вступи в сообщества, чтобы видеть их публикации здесь' : 'Вступи в сообщества, чтобы видеть публикации'}
                            action={
                                <Link to="/communities">
                                    <Button className="nexus-gradient border-0 text-white rounded-xl shadow-nexus">
                                        Найти сообщества
                                    </Button>
                                </Link>
                            }
                        />
                    ) : (
                        <div className="nexus-feed-shell">
                            {posts.map(post => (
                                <PostCard key={post.id} post={post} currentUser={user} />
                            ))}
                        </div>
                    )}
                </div>

                {/* Right sidebar (desktop) */}
                <div className="hidden lg:flex flex-col gap-4 w-72 flex-shrink-0">
                    <div className="nexus-card p-4">
                        <div className="flex items-center gap-2 mb-3">
                            <TrendingUp className="w-4 h-4 text-primary" />
                            <h3 className="font-bold text-sm">Популярные сообщества</h3>
                        </div>
                        <div className="flex flex-col">
                            {communities.map((c, i) => (
                                <Link key={c.id} to={`/community/${c.id}`}>
                                    <div className="flex items-center gap-2.5 py-2 hover:bg-muted/50 transition-colors px-1 rounded-xl">
                                        <span className="text-xs font-black text-muted-foreground w-4">{i + 1}</span>
                                        <img
                                            src={c.avatar_url || `https://api.dicebear.com/7.x/shapes/svg?seed=${c.name}`}
                                            className="w-8 h-8 rounded-xl object-cover"
                                            alt=""
                                        />
                                        <div className="flex-1 min-w-0">
                                            <p className="text-xs font-bold truncate">{c.name}</p>
                                            <p className="text-[10px] text-muted-foreground">{c.member_count || 0} участников</p>
                                        </div>
                                    </div>
                                    {i < communities.length - 1 && <div className="border-b border-border/30 mx-1" />}
                                </Link>
                            ))}
                        </div>
                        <Link to="/communities" className="block mt-3">
                            <Button variant="ghost" size="sm" className="w-full text-xs rounded-xl">
                                Все сообщества
                            </Button>
                        </Link>
                    </div>

                    {user && (
                        <div className="nexus-card p-4">
                            <div className="flex items-center gap-2 mb-2">
                                <Users className="w-4 h-4 text-primary" />
                                <h3 className="font-bold text-sm">Создай сообщество</h3>
                            </div>
                            <p className="text-xs text-muted-foreground mb-3">
                                Объедини людей по интересам — аниме, игры, ролевые игры или любое другое хобби.
                            </p>
                            <Link to="/create-community">
                                <Button className="w-full nexus-gradient border-0 text-white rounded-xl shadow-nexus text-xs h-8">
                                    Создать
                                </Button>
                            </Link>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
