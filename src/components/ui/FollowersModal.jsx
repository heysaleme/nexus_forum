import { useState, useEffect } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Link } from 'react-router-dom';
import { Search, Users, UserCheck } from 'lucide-react';
import { nexusApi } from '@/api/nexusApi';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';

function UserRow({ u, onClick }) {
    return (
        <Link
            to={`/user/${u.id}`}
            onClick={onClick}
            className="flex items-center gap-3 p-3 hover:bg-muted/40 rounded-xl transition-colors"
        >
            <img
                src={u.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${u.email || u.id}`}
                className="w-10 h-10 rounded-full object-cover flex-shrink-0"
                alt=""
            />
            <div className="flex-1 min-w-0">
                <p className="text-sm font-bold truncate">{u.username || u.full_name}</p>
                {u.bio && <p className="text-xs text-muted-foreground truncate">{u.bio}</p>}
            </div>
        </Link>
    );
}

export default function FollowersModal({ open, onClose, userId, defaultTab = 'followers' }) {
    const [followers, setFollowers] = useState([]);
    const [following, setFollowing] = useState([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState('');

    useEffect(() => {
        if (open && userId) {
            loadData();
        }
    }, [open, userId]);

    const loadData = async () => {
        setLoading(true);
        const [followersData, followingData] = await Promise.all([
            nexusApi.entities.UserFollow.filter({ following_id: userId }).catch(() => []),
            nexusApi.entities.UserFollow.filter({ follower_id: userId }).catch(() => []),
        ]);

        // If the API returns full user objects, great; if not, fetch them
        const resolveUsers = async (list, idField) => {
            if (list.length === 0) return [];
            if (list[0]?.username) return list; // Already resolved to users
            return Promise.all(
                list.map(f =>
                    nexusApi.entities.User.filter({ id: f[idField] })
                        .then(r => r[0])
                        .catch(() => null)
                )
            ).then(r => r.filter(Boolean));
        };

        const resolvedFollowers = await resolveUsers(followersData, 'follower_id');
        const resolvedFollowing = await resolveUsers(followingData, 'following_id');

        setFollowers(resolvedFollowers);
        setFollowing(resolvedFollowing);
        setLoading(false);
    };

    const filterUsers = (list) => {
        if (!search.trim()) return list;
        const q = search.toLowerCase();
        return list.filter(u =>
            (u.username || '').toLowerCase().includes(q) ||
            (u.full_name || '').toLowerCase().includes(q)
        );
    };

    return (
        <Dialog open={open} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md rounded-2xl p-5 bg-card border border-border">
                <DialogHeader>
                    <DialogTitle className="font-display font-black text-base">
                        Подписчики и подписки
                    </DialogTitle>
                </DialogHeader>

                <div className="relative mt-1">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                    <Input
                        value={search}
                        onChange={e => setSearch(e.target.value)}
                        placeholder="Поиск..."
                        className="pl-9 rounded-xl border-border/50 h-9 text-sm"
                    />
                </div>

                {loading ? (
                    <LoadingSpinner className="py-8" />
                ) : (
                    <Tabs defaultValue={defaultTab} className="mt-2">
                        <TabsList className="bg-muted/50 rounded-xl p-1 w-full">
                            <TabsTrigger value="followers" className="flex-1 rounded-lg text-xs gap-1.5">
                                <Users className="w-3.5 h-3.5" />
                                Подписчики ({followers.length})
                            </TabsTrigger>
                            <TabsTrigger value="following" className="flex-1 rounded-lg text-xs gap-1.5">
                                <UserCheck className="w-3.5 h-3.5" />
                                Подписки ({following.length})
                            </TabsTrigger>
                        </TabsList>

                        <TabsContent value="followers">
                            <div className="max-h-72 overflow-y-auto mt-2 space-y-0.5">
                                {filterUsers(followers).length === 0 ? (
                                    <EmptyState icon={Users} title="Нет подписчиков" className="py-6" />
                                ) : (
                                    filterUsers(followers).map(u => (
                                        <UserRow key={u.id} u={u} onClick={onClose} />
                                    ))
                                )}
                            </div>
                        </TabsContent>

                        <TabsContent value="following">
                            <div className="max-h-72 overflow-y-auto mt-2 space-y-0.5">
                                {filterUsers(following).length === 0 ? (
                                    <EmptyState icon={UserCheck} title="Нет подписок" className="py-6" />
                                ) : (
                                    filterUsers(following).map(u => (
                                        <UserRow key={u.id} u={u} onClick={onClose} />
                                    ))
                                )}
                            </div>
                        </TabsContent>
                    </Tabs>
                )}
            </DialogContent>
        </Dialog>
    );
}
