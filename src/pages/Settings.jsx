import { useState, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Settings as SettingsIcon, User, Lock, Palette, Upload, LogOut } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import { useNavigate } from 'react-router-dom';
import { validateFileSize, UPLOAD_LIMITS_MB, limitLabelForCategory } from '@/lib/validateFileSize';

const THEMES = [
    { id: 'default', label: 'По умолчанию', gradient: 'from-primary/30 to-accent/30' },
    { id: 'purple', label: 'Фиолетовый', gradient: 'from-purple-400/40 to-indigo-500/40' },
    { id: 'ocean', label: 'Океан', gradient: 'from-blue-400/40 to-cyan-400/40' },
    { id: 'sunset', label: 'Закат', gradient: 'from-orange-400/40 to-pink-500/40' },
    { id: 'forest', label: 'Лес', gradient: 'from-green-400/40 to-emerald-500/40' },
    { id: 'dark', label: 'Тёмный', gradient: 'from-gray-600/50 to-gray-900/50' },
];

export default function Settings() {
    const { user, checkUserAuth } = useAuth();
    const { toast } = useToast();
    const navigate = useNavigate();
    const [saving, setSaving] = useState(false);
    const [uploading, setUploading] = useState(false);

    const [profile, setProfile] = useState({
        username: '', bio: '', avatar_url: '', banner_url: '',
        profile_theme: 'default', title: '', allow_dms: true, is_private: false,
        email_notify_reply: true, email_notify_mention: true, email_notify_follow: true,
        email_notify_moderation: true, email_notify_report: true,
    });

    const [pendingRequests, setPendingRequests] = useState([]);
    const [loadingRequests, setLoadingRequests] = useState(false);

    const [oldPassword, setOldPassword] = useState('');
    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [changingPassword, setChangingPassword] = useState(false);
    const [sessions, setSessions] = useState([]);
    const [loadingSessions, setLoadingSessions] = useState(false);

    const loadPendingRequests = async () => {
        if (!user || !user.is_private) return;
        setLoadingRequests(true);
        try {
            const data = await nexusApi.entities.UserFollow.getPendingRequests();
            setPendingRequests(data || []);
        } catch (err) {
            console.error('Failed to load pending requests:', err);
        }
        setLoadingRequests(false);
    };

    useEffect(() => {
        if (user) {
            setProfile({
                username: user.username || user.full_name || '',
                bio: user.bio || '',
                avatar_url: user.avatar_url || '',
                banner_url: user.banner_url || '',
                profile_theme: user.profile_theme || 'default',
                title: user.title || '',
                allow_dms: user.allow_dms !== false,
                is_private: user.is_private || false,
                email_notify_reply: user.email_notify_reply !== false,
                email_notify_mention: user.email_notify_mention !== false,
                email_notify_follow: user.email_notify_follow !== false,
                email_notify_moderation: user.email_notify_moderation !== false,
                email_notify_report: user.email_notify_report !== false,
            });
            if (user.is_private) {
                loadPendingRequests();
            }
            loadSessions();
        }
    }, [user]);

    const loadSessions = async () => {
        if (!user) return;
        setLoadingSessions(true);
        try {
            const data = await nexusApi.auth.listSessions();
            setSessions(Array.isArray(data) ? data : []);
        } catch {
            setSessions([]);
        } finally {
            setLoadingSessions(false);
        }
    };

    const handleRevokeSession = async (sessionId) => {
        try {
            await nexusApi.auth.revokeSession(sessionId);
            toast({ title: 'Сессия завершена' });
            loadSessions();
        } catch {
            toast({ title: 'Не удалось завершить сессию', variant: 'destructive' });
        }
    };

    const handleRevokeOtherSessions = async () => {
        try {
            await nexusApi.auth.revokeOtherSessions();
            toast({ title: 'Другие устройства отключены' });
            loadSessions();
        } catch {
            toast({ title: 'Не удалось отключить устройства', variant: 'destructive' });
        }
    };

    const handleAcceptRequest = async (followerId) => {
        try {
            await nexusApi.entities.UserFollow.acceptRequest(followerId);
            setPendingRequests(prev => prev.filter(r => r.id !== followerId));
            toast({ title: '✅ Запрос принят' });
        } catch (err) {
            toast({ title: 'Не удалось принять запрос', variant: 'destructive' });
        }
    };

    const handleRejectRequest = async (followerId) => {
        try {
            await nexusApi.entities.UserFollow.rejectRequest(followerId);
            setPendingRequests(prev => prev.filter(r => r.id !== followerId));
            toast({ title: '❌ Запрос отклонен' });
        } catch (err) {
            toast({ title: 'Не удалось отклонить запрос', variant: 'destructive' });
        }
    };

    const handleChangePassword = async () => {
        if (newPassword.length < 6) {
            toast({ title: 'Новый пароль должен быть не менее 6 символов', variant: 'destructive' });
            return;
        }
        if (newPassword !== confirmPassword) {
            toast({ title: 'Пароли не совпадают', variant: 'destructive' });
            return;
        }
        setChangingPassword(true);
        try {
            await nexusApi.auth.changePassword({ old_password: oldPassword, new_password: newPassword });
            toast({ title: '✅ Пароль успешно изменен!' });
            setOldPassword('');
            setNewPassword('');
            setConfirmPassword('');
        } catch (err) {
            toast({ title: err.error || 'Не удалось изменить пароль', variant: 'destructive' });
        }
        setChangingPassword(false);
    };

    const handleSave = async () => {
        setSaving(true);
        try {
            await nexusApi.auth.updateMe(profile);
            if (checkUserAuth) {
                await checkUserAuth();
            }
            toast({ title: '✅ Настройки сохранены!' });
        } catch (err) {
            toast({ title: 'Ошибка при сохранении настроек', variant: 'destructive' });
        }
        setSaving(false);
    };

    const handleAvatarUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setUploading(true);
        try {
            validateFileSize(file, UPLOAD_LIMITS_MB['profile/avatars']);
            const { file_url } = await nexusApi.integrations.Core.UploadFile({ file, category: 'profile/avatars' });
            setProfile(prev => ({ ...prev, avatar_url: file_url }));
            toast({ title: '✅ Аватар загружен' });
        } catch (err) {
            toast({ title: err.message || 'Не удалось загрузить аватар', variant: 'destructive' });
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    };

    const handleBannerUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setUploading(true);
        try {
            validateFileSize(file, UPLOAD_LIMITS_MB['profile/banners']);
            const { file_url } = await nexusApi.integrations.Core.UploadFile({ file, category: 'profile/banners' });
            setProfile(prev => ({ ...prev, banner_url: file_url }));
            toast({ title: '✅ Баннер загружен' });
        } catch (err) {
            toast({ title: err.message || 'Не удалось загрузить баннер', variant: 'destructive' });
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    };

    const handleLogout = () => {
        nexusApi.auth.logout('/');
    };

    return (
        <div className="max-w-2xl mx-auto px-4 py-4">
            <div className="flex items-center gap-2 mb-5">
                <SettingsIcon className="w-5 h-5 text-primary" />
                <h1 className="text-xl font-display font-black">Настройки</h1>
            </div>

            <Tabs defaultValue="profile">
                <TabsList className="bg-muted/50 rounded-xl p-1 mb-4">
                    <TabsTrigger value="profile" className="rounded-lg text-xs gap-1.5"><User className="w-3.5 h-3.5" />Профиль</TabsTrigger>
                    <TabsTrigger value="theme" className="rounded-lg text-xs gap-1.5"><Palette className="w-3.5 h-3.5" />Оформление</TabsTrigger>
                    <TabsTrigger value="privacy" className="rounded-lg text-xs gap-1.5"><Lock className="w-3.5 h-3.5" />Приватность и email</TabsTrigger>
                </TabsList>

                {/* Profile tab */}
                <TabsContent value="profile">
                    <div className="space-y-4">
                        {/* Avatar & Banner */}
                        <div className="nexus-card overflow-hidden">
                            <div className="relative h-28 bg-gradient-to-br from-primary/20 to-accent/20">
                                {profile.banner_url && <img src={profile.banner_url} className="w-full h-full object-cover" alt="" />}
                                <label className="absolute inset-0 flex items-center justify-center bg-black/30 opacity-0 hover:opacity-100 transition-opacity cursor-pointer">
                                    <Upload className="w-6 h-6 text-white" />
                                    <input type="file" accept="image/*" className="hidden" onChange={handleBannerUpload} />
                                </label>
                            </div>
                            <div className="p-4 flex items-center gap-4 -mt-8">
                                <label className="relative cursor-pointer">
                                    <img
                                        src={profile.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user?.email}`}
                                        className="w-16 h-16 rounded-2xl border-4 border-card object-cover"
                                        alt=""
                                    />
                                    <div className="absolute inset-0 flex items-center justify-center bg-black/40 rounded-2xl opacity-0 hover:opacity-100 transition-opacity">
                                        {uploading ? <LoadingSpinner size="sm" /> : <Upload className="w-4 h-4 text-white" />}
                                    </div>
                                    <input type="file" accept="image/*" className="hidden" onChange={handleAvatarUpload} />
                                </label>
                                <p className="text-xs text-muted-foreground pt-6">
                                    Аватар {limitLabelForCategory('profile/avatars')}, баннер {limitLabelForCategory('profile/banners')}
                                </p>
                            </div>
                        </div>

                        <div className="nexus-card p-4 space-y-4">
                            <div>
                                <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Никнейм</Label>
                                <Input value={profile.username} onChange={e => setProfile(p => ({ ...p, username: e.target.value }))} className="rounded-xl border-border/50 h-10 text-sm" />
                            </div>
                            <div>
                                <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Пользовательский титул</Label>
                                <Input value={profile.title} onChange={e => setProfile(p => ({ ...p, title: e.target.value }))} placeholder="например: Художник · Аниматор" className="rounded-xl border-border/50 h-10 text-sm" />
                            </div>
                            <div>
                                <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">О себе</Label>
                                <Textarea value={profile.bio} onChange={e => setProfile(p => ({ ...p, bio: e.target.value }))} placeholder="Расскажи о себе..." className="rounded-xl border-border/50 text-sm min-h-20 resize-none" />
                            </div>
                            <Button onClick={handleSave} disabled={saving} className="w-full nexus-gradient border-0 text-white rounded-xl h-10 font-bold shadow-nexus">
                                {saving ? <LoadingSpinner size="sm" /> : 'Сохранить'}
                            </Button>
                        </div>
                    </div>
                </TabsContent>

                {/* Theme tab */}
                <TabsContent value="theme">
                    <div className="nexus-card p-4">
                        <h3 className="font-bold text-sm mb-3">Тема профиля</h3>
                        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                            {THEMES.map(theme => (
                                <button
                                    key={theme.id}
                                    onClick={() => setProfile(p => ({ ...p, profile_theme: theme.id }))}
                                    className={`h-20 rounded-2xl bg-gradient-to-br ${theme.gradient} flex items-end p-2.5 transition-all ${profile.profile_theme === theme.id ? 'ring-2 ring-primary shadow-nexus scale-105' : 'hover:scale-102 hover:shadow-md'
                                        }`}
                                >
                                    <span className="text-xs font-bold text-foreground/80">{theme.label}</span>
                                </button>
                            ))}
                        </div>
                        <Button onClick={handleSave} disabled={saving} className="mt-4 w-full nexus-gradient border-0 text-white rounded-xl h-10 font-bold shadow-nexus">
                            {saving ? <LoadingSpinner size="sm" /> : 'Применить'}
                        </Button>
                    </div>
                </TabsContent>

                {/* Privacy tab */}
                <TabsContent value="privacy">
                    <div className="nexus-card p-4 space-y-4">
                        {[
                            { key: 'is_private', label: 'Приватный профиль', desc: 'Только подписчики видят твои посты' },
                            { key: 'allow_dms', label: 'Личные сообщения', desc: 'Разрешить всем отправлять тебе сообщения' },
                        ].map(({ key, label, desc }) => (
                            <div key={key} className="flex items-center justify-between py-2">
                                <div>
                                    <p className="text-sm font-semibold">{label}</p>
                                    <p className="text-xs text-muted-foreground">{desc}</p>
                                </div>
                                <Switch
                                    checked={profile[key]}
                                    onCheckedChange={val => setProfile(p => ({ ...p, [key]: val }))}
                                />
                            </div>
                        ))}
                        <div className="border-t border-border/40 pt-4 space-y-3">
                            <h4 className="text-sm font-bold">Email-уведомления</h4>
                            <p className="text-xs text-muted-foreground">Требуется настройка SMTP на сервере (см. backend/.env.example)</p>
                            {[
                                { key: 'email_notify_reply', label: 'Ответы на комментарии' },
                                { key: 'email_notify_mention', label: 'Упоминания' },
                                { key: 'email_notify_follow', label: 'Новые подписчики' },
                                { key: 'email_notify_moderation', label: 'Ответы модераторов' },
                                { key: 'email_notify_report', label: 'Решённые жалобы' },
                            ].map(({ key, label }) => (
                                <div key={key} className="flex items-center justify-between py-1">
                                    <p className="text-sm">{label}</p>
                                    <Switch
                                        checked={profile[key]}
                                        onCheckedChange={val => setProfile(p => ({ ...p, [key]: val }))}
                                    />
                                </div>
                            ))}
                        </div>
                        <Button onClick={handleSave} disabled={saving} className="w-full nexus-gradient border-0 text-white rounded-xl h-10 font-bold shadow-nexus">
                            {saving ? <LoadingSpinner size="sm" /> : 'Сохранить'}
                        </Button>
                    </div>

                    {user?.is_private && (
                        <div className="nexus-card p-4 mt-4 space-y-3">
                            <h3 className="font-bold text-sm mb-1 flex items-center justify-between">
                                <span>Запросы на подписку</span>
                                {pendingRequests.length > 0 && (
                                    <span className="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded-full font-semibold">
                                        {pendingRequests.length}
                                    </span>
                                )}
                            </h3>
                            {loadingRequests ? (
                                <div className="py-4 text-center"><LoadingSpinner size="sm" /></div>
                            ) : pendingRequests.length === 0 ? (
                                <p className="text-xs text-muted-foreground py-2 text-center">Нет новых запросов на подписку</p>
                            ) : (
                                <div className="space-y-3 max-h-60 overflow-y-auto pr-1">
                                    {pendingRequests.map(reqUser => (
                                        <div key={reqUser.id} className="flex items-center justify-between p-2 rounded-xl bg-muted/30 border border-border/20">
                                            <div className="flex items-center gap-2.5">
                                                <img
                                                    src={reqUser.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${reqUser.email}`}
                                                    className="w-8 h-8 rounded-full object-cover"
                                                    alt=""
                                                />
                                                <div>
                                                    <p className="text-xs font-bold leading-tight">
                                                        {reqUser.full_name || reqUser.username}
                                                    </p>
                                                    <p className="text-[10px] text-muted-foreground">
                                                        @{reqUser.username}
                                                    </p>
                                                </div>
                                            </div>
                                            <div className="flex gap-1.5">
                                                <Button
                                                    size="sm"
                                                    onClick={() => handleAcceptRequest(reqUser.id)}
                                                    className="h-7 px-2.5 text-[10px] font-bold rounded-lg bg-primary text-white border-0 hover:bg-primary/95"
                                                >
                                                    Принять
                                                </Button>
                                                <Button
                                                    size="sm"
                                                    variant="outline"
                                                    onClick={() => handleRejectRequest(reqUser.id)}
                                                    className="h-7 px-2.5 text-[10px] font-bold rounded-lg border-border"
                                                >
                                                    Отклонить
                                                </Button>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    )}

                    <div className="nexus-card p-4 mt-4 space-y-3">
                        <h3 className="font-bold text-sm mb-1">Активные сессии</h3>
                        {loadingSessions ? (
                            <LoadingSpinner size="sm" className="py-4" />
                        ) : sessions.length === 0 ? (
                            <p className="text-xs text-muted-foreground">Нет активных сессий</p>
                        ) : (
                            <div className="space-y-2 max-h-48 overflow-y-auto">
                                {sessions.map((s) => (
                                    <div key={s.id} className="flex items-start justify-between gap-2 p-2 rounded-xl bg-muted/30 text-xs">
                                        <div className="min-w-0">
                                            <p className="font-semibold truncate">{s.user_agent || 'Неизвестное устройство'}</p>
                                            <p className="text-muted-foreground">{s.ip_address || 'IP неизвестен'}</p>
                                            <p className="text-[10px] text-muted-foreground">до {new Date(s.expires_at).toLocaleString()}</p>
                                        </div>
                                        <Button size="sm" variant="outline" className="h-7 text-[10px] shrink-0" onClick={() => handleRevokeSession(s.id)}>
                                            Завершить
                                        </Button>
                                    </div>
                                ))}
                            </div>
                        )}
                        {sessions.length > 1 && (
                            <Button variant="outline" size="sm" className="w-full rounded-xl h-8 text-xs" onClick={handleRevokeOtherSessions}>
                                Выйти на всех других устройствах
                            </Button>
                        )}
                    </div>

                    <div className="nexus-card p-4 mt-4 space-y-4">
                        <h3 className="font-bold text-sm mb-1">Смена пароля</h3>
                        <div className="space-y-3">
                            <div>
                                <Label className="text-xs font-semibold text-muted-foreground mb-1 block">Текущий пароль</Label>
                                <Input
                                    type="password"
                                    value={oldPassword}
                                    onChange={e => setOldPassword(e.target.value)}
                                    placeholder="Введите старый пароль..."
                                    className="rounded-xl border-border/50 h-9 text-xs"
                                />
                            </div>
                            <div>
                                <Label className="text-xs font-semibold text-muted-foreground mb-1 block">Новый пароль</Label>
                                <Input
                                    type="password"
                                    value={newPassword}
                                    onChange={e => setNewPassword(e.target.value)}
                                    placeholder="Минимум 6 символов..."
                                    className="rounded-xl border-border/50 h-9 text-xs"
                                />
                            </div>
                            <div>
                                <Label className="text-xs font-semibold text-muted-foreground mb-1 block">Подтвердите новый пароль</Label>
                                <Input
                                    type="password"
                                    value={confirmPassword}
                                    onChange={e => setConfirmPassword(e.target.value)}
                                    placeholder="Повторите новый пароль..."
                                    className="rounded-xl border-border/50 h-9 text-xs"
                                />
                            </div>
                            <Button
                                onClick={handleChangePassword}
                                disabled={changingPassword || !oldPassword || !newPassword || !confirmPassword}
                                className="w-full nexus-gradient border-0 text-white rounded-xl h-9 text-xs font-bold shadow-nexus"
                            >
                                {changingPassword ? <LoadingSpinner size="sm" /> : 'Обновить пароль'}
                            </Button>
                        </div>
                    </div>

                    <div className="nexus-card p-4 mt-4">
                        <h3 className="font-bold text-sm text-destructive mb-2">Выход из аккаунта</h3>
                        <Button onClick={handleLogout} variant="outline" className="w-full rounded-xl h-10 text-sm gap-2 text-destructive border-destructive/30 hover:bg-destructive/10">
                            <LogOut className="w-4 h-4" />Выйти из аккаунта
                        </Button>
                    </div>
                </TabsContent>
            </Tabs>
        </div>
    );
}