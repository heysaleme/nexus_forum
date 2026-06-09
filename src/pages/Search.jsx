import { useState, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useSearchParams, Link } from 'react-router-dom';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import PostCard from '@/components/feed/PostCard';
import { Search as SearchIcon, User, Users, FileText } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useAuth } from '@/lib/AuthContext';
import { profilePath } from '@/lib/profileLink';
import { motion } from 'framer-motion';

const TABS = [
    { id: 'all', label: 'Всё', icon: SearchIcon },
    { id: 'posts', label: 'Посты', icon: FileText },
    { id: 'communities', label: 'Сообщества', icon: Users },
    { id: 'users', label: 'Пользователи', icon: User },
];

export default function Search() {
    const { user } = useAuth();
    const [searchParams, setSearchParams] = useSearchParams();
    const [query, setQuery] = useState(searchParams.get('q') || '');
    const [activeTab, setActiveTab] = useState('all');
    const [results, setResults] = useState({ posts: [], communities: [], users: [] });
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        const q = searchParams.get('q');
        if (q) { setQuery(q); doSearch(q); }
    }, [searchParams]);

    const doSearch = async (q) => {
        if (!q.trim()) return;
        setLoading(true);
        try {
            const data = await nexusApi.Search.query(q);
            setResults({
                posts: data.posts || [],
                communities: data.communities || [],
                users: data.users || [],
            });
        } catch (err) {
            console.error('Search failed:', err);
            setResults({ posts: [], communities: [], users: [] });
        }
        setLoading(false);
    };

    const handleSearch = (e) => {
        e.preventDefault();
        if (query.trim()) {
            setSearchParams({ q: query });
            doSearch(query);
        }
    };

    const total = results.posts.length + results.communities.length + results.users.length;

    return (
        <div className="max-w-3xl mx-auto px-4 py-4">
            <div className="flex items-center gap-2 mb-4">
                <SearchIcon className="w-5 h-5 text-primary" />
                <h1 className="text-xl font-display font-black">Поиск</h1>
            </div>

            <form onSubmit={handleSearch} className="mb-4">
                <div className="relative">
                    <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                    <Input
                        value={query}
                        onChange={e => setQuery(e.target.value)}
                        placeholder="Поиск постов, сообществ, пользователей..."
                        className="pl-9 bg-muted/50 border-0 rounded-xl h-11 text-sm"
                    />
                </div>
            </form>

            {/* Tabs */}
            <div className="flex gap-2 overflow-x-auto scrollbar-hide mb-4 pb-1">
                {TABS.map(({ id, label, icon: Icon }) => (
                    <button
                        key={id}
                        onClick={() => setActiveTab(id)}
                        className={`flex items-center gap-1.5 px-3.5 py-1.5 rounded-xl text-sm font-semibold flex-shrink-0 whitespace-nowrap transition-all ${activeTab === id ? 'nexus-gradient text-white shadow-nexus' : 'bg-muted/60 text-muted-foreground hover:bg-muted'
                            }`}
                    >
                        <Icon className="w-3.5 h-3.5" />
                        {label}
                    </button>
                ))}
            </div>

            {loading ? (
                <LoadingSpinner size="lg" className="py-20" />
            ) : !query ? (
                <EmptyState icon={SearchIcon} title="Введите запрос" description="Ищи посты, сообщества и пользователей" />
            ) : total === 0 ? (
                <EmptyState icon={SearchIcon} title={`Ничего не найдено по "${query}"`} description="Попробуй изменить запрос" />
            ) : (
                <div className="space-y-5">
                    {/* Posts */}
                    {(activeTab === 'all' || activeTab === 'posts') && results.posts.length > 0 && (
                        <div>
                            {activeTab === 'all' && (
                                <h3 className="text-sm font-bold text-muted-foreground mb-2 flex items-center gap-1.5">
                                    <FileText className="w-3.5 h-3.5" />Публикации ({results.posts.length})
                                </h3>
                            )}
                            <div className="space-y-3">
                                {results.posts.slice(0, activeTab === 'all' ? 3 : 20).map(post => (
                                    <PostCard key={post.id} post={post} currentUser={user} />
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Communities */}
                    {(activeTab === 'all' || activeTab === 'communities') && results.communities.length > 0 && (
                        <div>
                            {activeTab === 'all' && (
                                <h3 className="text-sm font-bold text-muted-foreground mb-2 flex items-center gap-1.5">
                                    <Users className="w-3.5 h-3.5" />Сообщества ({results.communities.length})
                                </h3>
                            )}
                            <div className="space-y-2">
                                {results.communities.slice(0, activeTab === 'all' ? 4 : 20).map(c => (
                                    <Link key={c.id} to={`/community/${c.id}`}>
                                        <motion.div whileHover={{ x: 2 }} className="nexus-card nexus-card-hover p-3 flex items-center gap-3">
                                            <img src={c.avatar_url || `https://api.dicebear.com/7.x/shapes/svg?seed=${c.name}`} className="w-10 h-10 rounded-xl object-cover" alt="" />
                                            <div className="flex-1 min-w-0">
                                                <p className="text-sm font-bold truncate">{c.name}</p>
                                                <p className="text-xs text-muted-foreground line-clamp-1">{c.description}</p>
                                            </div>
                                            <div className="text-right flex-shrink-0">
                                                <p className="text-xs font-bold text-primary">{c.member_count || 0}</p>
                                                <p className="text-[10px] text-muted-foreground">участников</p>
                                            </div>
                                        </motion.div>
                                    </Link>
                                ))}
                            </div>
                        </div>
                    )}

                    {/* Users */}
                    {(activeTab === 'all' || activeTab === 'users') && results.users.length > 0 && (
                        <div>
                            {activeTab === 'all' && (
                                <h3 className="text-sm font-bold text-muted-foreground mb-2 flex items-center gap-1.5">
                                    <User className="w-3.5 h-3.5" />Пользователи ({results.users.length})
                                </h3>
                            )}
                            <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                                {results.users.slice(0, activeTab === 'all' ? 6 : 30).map(u => (
                                    <Link key={u.id} to={profilePath(u.id, user?.id)}>
                                        <motion.div whileHover={{ y: -2 }} className="nexus-card nexus-card-hover p-3 flex flex-col items-center gap-2 text-center cursor-pointer">
                                            <img
                                                src={u.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${u.username}`}
                                                className="w-12 h-12 rounded-full object-cover ring-2 ring-primary/20"
                                                alt=""
                                            />
                                            <div className="w-full">
                                                <p className="text-sm font-bold truncate">{u.username}</p>
                                                {u.role && u.role !== 'user' && (
                                                    <Badge className={`text-[9px] px-1.5 py-0 mt-0.5 ${u.role === 'admin' ? 'bg-red-500/10 text-red-500 border-0' : 'bg-blue-500/10 text-blue-500 border-0'}`}>
                                                        {u.role === 'admin' ? 'Администратор' : 'Модератор'}
                                                    </Badge>
                                                )}
                                                {u.bio && (
                                                    <p className="text-[10px] text-muted-foreground line-clamp-1 mt-0.5">{u.bio}</p>
                                                )}
                                            </div>
                                        </motion.div>
                                    </Link>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}