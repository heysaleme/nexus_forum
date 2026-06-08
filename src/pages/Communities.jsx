import { useState, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import CommunityCard from '@/components/community/CommunityCard';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { Search, Compass, Plus } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';

const categories = [
    { id: 'all', label: 'Все' },
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

export default function Communities() {
    const { user } = useAuth();
    const { toast } = useToast();
    const [communities, setCommunities] = useState([]);
    const [loading, setLoading] = useState(true);
    const [category, setCategory] = useState('all');
    const [search, setSearch] = useState('');
    const [memberships, setMemberships] = useState(new Set());

    useEffect(() => {
        loadCommunities();
    }, []);

    useEffect(() => {
        if (user) loadMemberships();
    }, [user]);

    const loadCommunities = async () => {
        setLoading(true);
        const data = await nexusApi.entities.Community.list('-member_count', 50);
        setCommunities(data);
        setLoading(false);
    };

    const loadMemberships = async () => {
        const data = await nexusApi.entities.CommunityMember.filter({ user_id: user.id });
        setMemberships(new Set(data.map(m => m.community_id)));
    };

    const handleJoin = async (community) => {
        if (!user) {
            toast({ title: 'Войдите, чтобы вступить', variant: 'destructive' });
            return;
        }
        if (memberships.has(community.id)) {
            const members = await nexusApi.entities.CommunityMember.filter({ user_id: user.id, community_id: community.id });
            if (members[0]) await nexusApi.entities.CommunityMember.delete(community.id);
            setMemberships(prev => { const s = new Set(prev); s.delete(community.id); return s; });
            toast({ title: `Вы покинули ${community.name}` });
        } else {
            await nexusApi.entities.CommunityMember.create({ user_id: user.id, community_id: community.id, role: 'member' });
            setMemberships(prev => new Set([...prev, community.id]));
            toast({ title: `Вы вступили в ${community.name}! 🎉` });
        }
    };

    const filtered = communities.filter(c => {
        const matchCategory = category === 'all' || c.category === category;
        const matchSearch = !search || c.name.toLowerCase().includes(search.toLowerCase()) || c.description?.toLowerCase().includes(search.toLowerCase());
        return matchCategory && matchSearch;
    });

    return (
        <div className="max-w-6xl mx-auto px-4 py-4">
            {/* Header */}
            <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                    <Compass className="w-5 h-5 text-primary" />
                    <h1 className="text-xl font-display font-black">Сообщества</h1>
                </div>
                {user && (
                    <Link to="/create-community">
                        <Button size="sm" className="nexus-gradient border-0 text-white rounded-xl shadow-nexus gap-1.5 text-xs h-8">
                            <Plus className="w-3.5 h-3.5" />
                            Создать
                        </Button>
                    </Link>
                )}
            </div>

            {/* Search */}
            <div className="relative mb-4">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                <Input
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                    placeholder="Найти сообщество..."
                    className="pl-9 bg-muted/50 border-0 rounded-xl h-10 text-sm"
                />
            </div>

            {/* Categories */}
            <div className="flex gap-2 overflow-x-auto scrollbar-hide mb-5 pb-1">
                {categories.map(({ id, label }) => (
                    <motion.button
                        key={id}
                        whileTap={{ scale: 0.95 }}
                        onClick={() => setCategory(id)}
                        className={`px-3.5 py-1.5 rounded-xl text-sm font-semibold whitespace-nowrap flex-shrink-0 transition-all ${category === id
                            ? 'nexus-gradient text-white shadow-nexus'
                            : 'bg-muted/60 text-muted-foreground hover:bg-muted'
                            }`}
                    >
                        {label}
                    </motion.button>
                ))}
            </div>

            {loading ? (
                <LoadingSpinner size="lg" className="py-20" />
            ) : filtered.length === 0 ? (
                <EmptyState
                    icon={Compass}
                    title="Сообщества не найдены"
                    description="Попробуй изменить поиск или фильтр категории"
                />
            ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
                    {filtered.map(community => (
                        <CommunityCard
                            key={community.id}
                            community={community}
                            onJoin={handleJoin}
                            isJoined={memberships.has(community.id)}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}