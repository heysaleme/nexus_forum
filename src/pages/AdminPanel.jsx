import { useState, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import { useNavigate } from 'react-router-dom';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Shield, Users, AlertTriangle, BarChart2, Search, Ban, Eye, CheckCircle, XCircle, Trash2 } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { motion } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';

export default function AdminPanel() {
    const { user } = useAuth();
    const navigate = useNavigate();
    const { toast } = useToast();
    const [users, setUsers] = useState([]);
    const [communities, setCommunities] = useState([]);
    const [reports, setReports] = useState([]);
    const [dashboard, setDashboard] = useState(null);
    const [loading, setLoading] = useState(true);
    const [userSearch, setUserSearch] = useState('');

    useEffect(() => {
        if (!user || user.role !== 'admin') { navigate('/'); return; }
        loadData();
    }, [user]);

    const loadData = async () => {
        setLoading(true);
        try {
            const [usersData, communitiesData, reportsData, dashboardData] = await Promise.all([
                nexusApi.entities.User.list('-created_date', 50),
                nexusApi.entities.Community.list('-member_count', 30),
                nexusApi.entities.Report.list(),
                nexusApi.analytics.getDashboard(),
            ]);
            setUsers(usersData);
            setCommunities(communitiesData);
            setReports(Array.isArray(reportsData) ? reportsData : []);
            setDashboard(dashboardData);
        } catch (err) {
            toast({ title: err.message || 'Не удалось загрузить данные панели', variant: 'destructive' });
        } finally {
            setLoading(false);
        }
    };

    const handleBanUser = async (u) => {
        await nexusApi.entities.User.update(u.id, { is_banned: !u.is_banned });
        toast({ title: u.is_banned ? '✅ Пользователь разблокирован' : '🚫 Пользователь заблокирован' });
        loadData();
    };

    const handleChangeRole = async (u, role) => {
        await nexusApi.entities.User.update(u.id, { role });
        toast({ title: `Роль изменена на ${role}` });
        loadData();
    };

    const handleReport = async (report, status) => {
        await nexusApi.entities.Report.update(report.id, { status });
        toast({ title: `Жалоба помечена как "${status}"` });
        loadData();
    };

    const handleDeletePost = async (postId) => {
        await nexusApi.entities.Post.update(postId, { status: 'removed' });
        toast({ title: '🗑️ Публикация удалена' });
        loadData();
    };

    const handleDeleteCommunity = async (commId) => {
        if (!window.confirm('Вы уверены, что хотите удалить это сообщество? Все публикации сообщества будут удалены!')) return;
        try {
            await nexusApi.entities.Community.delete(commId);
            toast({ title: '🗑️ Сообщество удалено' });
            loadData();
        } catch (err) {
            toast({ title: 'Не удалось удалить сообщество', variant: 'destructive' });
        }
    };

    const reasonLabels = {
        spam: 'Спам',
        harassment: 'Харассмент',
        nsfw: 'NSFW',
        other: 'Другое',
    };

    const stats = {
        totalUsers: dashboard?.total_users ?? 0,
        dau: dashboard?.dau ?? 0,
        mau: dashboard?.mau ?? 0,
        bannedUsers: dashboard?.banned_users ?? 0,
        totalCommunities: dashboard?.total_communities ?? 0,
        pendingReports: dashboard?.pending_reports ?? 0,
        totalPosts: dashboard?.total_posts ?? 0,
        totalAdmins: dashboard?.total_admins ?? 0,
    };

    const chartData = (dashboard?.user_growth_30d || []).map((row) => ({
        name: row.day ? String(row.day).slice(5) : '',
        users: row.count || 0,
    }));

    const activity7dData = (dashboard?.activity_7d || []).map((row) => ({
        name: row.day ? String(row.day).slice(5) : '',
        users: row.users || 0,
        posts: row.posts || 0,
    }));

    const pieData = (dashboard?.report_reasons || []).map((row) => ({
        name: reasonLabels[row.reason] || row.reason || 'Неизвестно',
        value: row.count || 0,
    })).filter((item) => item.value > 0);
    const COLORS = ['#6A5AE0', '#8B7CFF', '#5EDFFF', '#EF4444', '#F59E0B', '#10B981'];

    const filteredUsers = users.filter(u =>
        !userSearch || u.full_name?.toLowerCase().includes(userSearch.toLowerCase()) || u.email?.toLowerCase().includes(userSearch.toLowerCase())
    );

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;

    return (
        <div className="max-w-7xl mx-auto px-4 py-4 overflow-x-hidden">
            <div className="flex items-center justify-between mb-5">
                <div className="flex items-center gap-2">
                    <div className="w-9 h-9 bg-destructive/10 rounded-xl flex items-center justify-center">
                        <Shield className="w-5 h-5 text-destructive" />
                    </div>
                    <h1 className="text-xl font-display font-black">Администрация</h1>
                </div>
                <Button 
                    onClick={() => navigate('/admin/reports')} 
                    variant="outline" 
                    className="rounded-xl h-8 text-xs font-bold gap-1.5 border-border"
                >
                    <Shield className="w-3.5 h-3.5 text-primary" />
                    Очередь модерации
                </Button>
            </div>

            {/* Stats grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-8 gap-3 mb-5">
                {[
                    { label: 'Пользователей', value: stats.totalUsers, color: 'text-primary', icon: Users },
                    { label: 'DAU', value: stats.dau, color: 'text-cyan-600', icon: BarChart2 },
                    { label: 'MAU', value: stats.mau, color: 'text-indigo-600', icon: BarChart2 },
                    { label: 'Заблокировано', value: stats.bannedUsers, color: 'text-destructive', icon: Ban },
                    { label: 'Сообществ', value: stats.totalCommunities, color: 'text-green-600', icon: Users },
                    { label: 'Жалоб', value: stats.pendingReports, color: 'text-orange-600', icon: AlertTriangle },
                    { label: 'Постов', value: stats.totalPosts, color: 'text-blue-600', icon: Eye },
                    { label: 'Администраторов', value: stats.totalAdmins, color: 'text-purple-600', icon: Shield },
                ].map(({ label, value, color, icon: Icon }) => (
                    <motion.div
                        key={label}
                        initial={{ opacity: 0, y: 12 }}
                        animate={{ opacity: 1, y: 0 }}
                        className="nexus-card p-4 text-center"
                    >
                        <Icon className={`w-5 h-5 mx-auto mb-1.5 ${color}`} />
                        <p className={`text-xl font-black ${color}`}>{value}</p>
                        <p className="text-[10px] text-muted-foreground leading-tight">{label}</p>
                    </motion.div>
                ))}
            </div>

            <Tabs defaultValue="stats">
                <TabsList className="bg-muted/50 rounded-xl p-1 mb-4 w-full flex flex-nowrap overflow-x-auto scrollbar-hide justify-start">
                    <TabsTrigger value="stats" className="rounded-lg text-xs gap-1.5 flex-shrink-0"><BarChart2 className="w-3.5 h-3.5" />Статистика</TabsTrigger>
                    <TabsTrigger value="users" className="rounded-lg text-xs gap-1.5 flex-shrink-0"><Users className="w-3.5 h-3.5" />Пользователи</TabsTrigger>
                    <TabsTrigger value="communities" className="rounded-lg text-xs gap-1.5 flex-shrink-0"><Users className="w-3.5 h-3.5" />Сообщества</TabsTrigger>
                    <TabsTrigger value="reports" className="rounded-lg text-xs gap-1.5 relative flex-shrink-0">
                        <AlertTriangle className="w-3.5 h-3.5" />
                        Жалобы
                        {stats.pendingReports > 0 && <span className="absolute -top-1 -right-1 w-3 h-3 bg-destructive rounded-full text-[8px] text-white flex items-center justify-center">{stats.pendingReports}</span>}
                    </TabsTrigger>
                </TabsList>

                {/* Statistics */}
                <TabsContent value="stats">
                    <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
                        <div className="nexus-card p-4 lg:col-span-2">
                            <h3 className="font-bold text-sm mb-3">Активность за неделю</h3>
                            {activity7dData.length === 0 ? (
                                <p className="text-xs text-muted-foreground py-16 text-center">Нет данных об активности за последние 7 дней</p>
                            ) : (
                            <ResponsiveContainer width="100%" height={200}>
                                <AreaChart data={activity7dData}>
                                    <defs>
                                        <linearGradient id="colorWeekUsers" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#6A5AE0" stopOpacity={0.3} />
                                            <stop offset="95%" stopColor="#6A5AE0" stopOpacity={0} />
                                        </linearGradient>
                                        <linearGradient id="colorWeekPosts" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#5EDFFF" stopOpacity={0.3} />
                                            <stop offset="95%" stopColor="#5EDFFF" stopOpacity={0} />
                                        </linearGradient>
                                    </defs>
                                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                                    <XAxis dataKey="name" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                                    <YAxis tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                                    <Tooltip contentStyle={{ background: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', fontSize: '12px' }} />
                                    <Area type="monotone" dataKey="users" stroke="#6A5AE0" strokeWidth={2} fill="url(#colorWeekUsers)" name="Новые пользователи" />
                                    <Area type="monotone" dataKey="posts" stroke="#5EDFFF" strokeWidth={2} fill="url(#colorWeekPosts)" name="Новые посты" />
                                </AreaChart>
                            </ResponsiveContainer>
                            )}
                        </div>

                        <div className="nexus-card p-4">
                            <h3 className="font-bold text-sm mb-3">Причины жалоб</h3>
                            {pieData.length === 0 ? (
                                <p className="text-xs text-muted-foreground py-16 text-center">Жалоб пока нет</p>
                            ) : (
                            <>
                            <ResponsiveContainer width="100%" height={180}>
                                <PieChart>
                                    <Pie data={pieData} cx="50%" cy="50%" innerRadius={50} outerRadius={70} dataKey="value">
                                        {pieData.map((entry, index) => <Cell key={index} fill={COLORS[index % COLORS.length]} />)}
                                    </Pie>
                                    <Tooltip contentStyle={{ background: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', fontSize: '12px' }} />
                                </PieChart>
                            </ResponsiveContainer>
                            <div className="space-y-1 mt-2">
                                {pieData.map((item, i) => (
                                    <div key={item.name} className="flex items-center gap-2 text-xs">
                                        <div className="w-2 h-2 rounded-full" style={{ background: COLORS[i % COLORS.length] }} />
                                        <span className="text-muted-foreground flex-1">{item.name}</span>
                                        <span className="font-bold">{item.value}</span>
                                    </div>
                                ))}
                            </div>
                            </>
                            )}
                        </div>

                        <div className="nexus-card p-4 lg:col-span-3">
                            <h3 className="font-bold text-sm mb-3">Рост пользователей (30 дней)</h3>
                            {chartData.length === 0 ? (
                                <p className="text-xs text-muted-foreground py-16 text-center">Нет данных о регистрациях за последние 30 дней</p>
                            ) : (
                            <ResponsiveContainer width="100%" height={200}>
                                <AreaChart data={chartData}>
                                    <defs>
                                        <linearGradient id="colorUsers" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#6A5AE0" stopOpacity={0.3} />
                                            <stop offset="95%" stopColor="#6A5AE0" stopOpacity={0} />
                                        </linearGradient>
                                        <linearGradient id="colorPosts" x1="0" y1="0" x2="0" y2="1">
                                            <stop offset="5%" stopColor="#5EDFFF" stopOpacity={0.3} />
                                            <stop offset="95%" stopColor="#5EDFFF" stopOpacity={0} />
                                        </linearGradient>
                                    </defs>
                                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                                    <XAxis dataKey="name" tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                                    <YAxis tick={{ fontSize: 11 }} stroke="hsl(var(--muted-foreground))" />
                                    <Tooltip contentStyle={{ background: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', fontSize: '12px' }} />
                                    <Area type="monotone" dataKey="users" stroke="#6A5AE0" strokeWidth={2} fill="url(#colorUsers)" name="Регистрации" />
                                </AreaChart>
                            </ResponsiveContainer>
                            )}
                        </div>
                    </div>
                </TabsContent>

                {/* Users */}
                <TabsContent value="users">
                    <div className="nexus-card">
                        <div className="p-4 border-b border-border/50">
                            <div className="relative">
                                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                                <Input value={userSearch} onChange={e => setUserSearch(e.target.value)} placeholder="Поиск пользователей..." className="pl-9 bg-muted/50 border-0 rounded-xl h-9 text-sm" />
                            </div>
                        </div>
                        <div className="divide-y divide-border/50">
                            {filteredUsers.map(u => (
                                <div key={u.id} className="flex items-center gap-3 p-3">
                                    <img src={u.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${u.email}`} className="w-9 h-9 rounded-full object-cover flex-shrink-0" alt="" />
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2">
                                            <p className="text-sm font-bold truncate">{u.full_name || 'Пользователь'}</p>
                                            {u.is_banned && <Badge className="bg-destructive/10 text-destructive text-[9px] border-0">Заблокирован</Badge>}
                                        </div>
                                        <p className="text-xs text-muted-foreground">{u.email}</p>
                                    </div>
                                    <div className="flex items-center gap-2 flex-shrink-0">
                                        <Select value={u.role || 'user'} onValueChange={role => handleChangeRole(u, role)}>
                                            <SelectTrigger className="h-7 w-28 text-xs rounded-lg border-border/50">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="user">Пользователь</SelectItem>
                                                <SelectItem value="moderator">Модератор</SelectItem>
                                                <SelectItem value="admin">Администратор</SelectItem>
                                            </SelectContent>
                                        </Select>
                                        <Button variant="ghost" size="sm" onClick={() => handleBanUser(u)} className={`h-7 w-7 p-0 rounded-lg ${u.is_banned ? 'text-green-600 hover:bg-green-100' : 'text-destructive hover:bg-destructive/10'}`}>
                                            {u.is_banned ? <CheckCircle className="w-4 h-4" /> : <Ban className="w-4 h-4" />}
                                        </Button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                </TabsContent>

                {/* Communities */}
                <TabsContent value="communities">
                    <div className="nexus-card divide-y divide-border/50">
                        {communities.map(c => (
                            <div key={c.id} className="flex items-center gap-3 p-3">
                                <img src={c.avatar_url || `https://api.dicebear.com/7.x/shapes/svg?seed=${c.name}`} className="w-9 h-9 rounded-xl object-cover flex-shrink-0" alt="" />
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-bold truncate">{c.name}</p>
                                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                        <span>{c.member_count || 0} уч.</span>
                                        <span className="capitalize">{c.category}</span>
                                        <Badge className={`text-[9px] border-0 ${c.is_private ? 'bg-muted text-muted-foreground' : 'bg-green-100 text-green-700'}`}>
                                            {c.is_private ? 'Приватное' : 'Публичное'}
                                        </Badge>
                                    </div>
                                </div>
                                <Button variant="ghost" size="sm" onClick={() => handleDeleteCommunity(c.id)} className="h-7 w-7 p-0 rounded-lg text-destructive hover:bg-destructive/10">
                                    <Trash2 className="w-4 h-4" />
                                </Button>
                            </div>
                        ))}
                    </div>
                </TabsContent>

                {/* Reports */}
                <TabsContent value="reports">
                    <div className="space-y-2">
                        {reports.length === 0 ? (
                            <div className="nexus-card p-8 text-center">
                                <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-2" />
                                <p className="text-sm font-bold">Нет жалоб!</p>
                            </div>
                        ) : reports.map(report => (
                            <div key={report.id} className="nexus-card p-4">
                                <div className="flex items-start justify-between gap-2">
                                    <div className="flex-1">
                                        <div className="flex items-center gap-2 mb-1">
                                            <Badge className={`text-[9px] border-0 ${report.status === 'pending' ? 'bg-orange-100 text-orange-700' :
                                                    report.status === 'resolved' ? 'bg-green-100 text-green-700' :
                                                        'bg-muted text-muted-foreground'
                                                }`}>{report.status}</Badge>
                                            <Badge variant="outline" className="text-[9px]">{report.reason}</Badge>
                                            <span className="text-[10px] text-muted-foreground">{report.target_type}</span>
                                        </div>
                                        <p className="text-xs text-muted-foreground">{report.description}</p>
                                        <p className="text-[10px] text-muted-foreground mt-1">От: {report.reporter_username}</p>
                                    </div>
                                    {report.status === 'pending' && (
                                        <div className="flex gap-1.5 flex-shrink-0">
                                            <Button size="sm" onClick={() => handleReport(report, 'resolved')} className="h-7 px-2.5 text-xs rounded-lg bg-green-100 text-green-700 hover:bg-green-200 border-0">
                                                <CheckCircle className="w-3.5 h-3.5 mr-1" />Решить
                                            </Button>
                                            <Button size="sm" onClick={() => handleReport(report, 'dismissed')} className="h-7 px-2.5 text-xs rounded-lg bg-muted text-muted-foreground hover:bg-muted/80 border-0">
                                                <XCircle className="w-3.5 h-3.5 mr-1" />Отклонить
                                            </Button>
                                        </div>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                </TabsContent>
            </Tabs>
        </div>
    );
}
