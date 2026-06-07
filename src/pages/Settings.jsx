import { useState, useEffect } from 'react';
import { base44 } from '@/api/base44Client';
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

const THEMES = [
    { id: 'default', label: 'По умолчанию', gradient: 'from-primary/30 to-accent/30' },
    { id: 'purple', label: 'Фиолетовый', gradient: 'from-purple-400/40 to-indigo-500/40' },
    { id: 'ocean', label: 'Океан', gradient: 'from-blue-400/40 to-cyan-400/40' },
    { id: 'sunset', label: 'Закат', gradient: 'from-orange-400/40 to-pink-500/40' },
    { id: 'forest', label: 'Лес', gradient: 'from-green-400/40 to-emerald-500/40' },
    { id: 'dark', label: 'Тёмный', gradient: 'from-gray-600/50 to-gray-900/50' },
];

export default function Settings() {
    const { user } = useAuth();
    const { toast } = useToast();
    const navigate = useNavigate();
    const [saving, setSaving] = useState(false);
    const [uploading, setUploading] = useState(false);

    const [profile, setProfile] = useState({
        username: '', bio: '', avatar_url: '', banner_url: '',
        profile_theme: 'default', title: '', allow_dms: true, is_private: false,
    });

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
            });
        }
    }, [user]);

    const handleSave = async () => {
        setSaving(true);
        await base44.auth.updateMe(profile);
        toast({ title: '✅ Настройки сохранены!' });
        setSaving(false);
    };

    const handleAvatarUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setUploading(true);
        const { file_url } = await base44.integrations.Core.UploadFile({ file });
        setProfile(prev => ({ ...prev, avatar_url: file_url }));
        setUploading(false);
    };

    const handleBannerUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setUploading(true);
        const { file_url } = await base44.integrations.Core.UploadFile({ file });
        setProfile(prev => ({ ...prev, banner_url: file_url }));
        setUploading(false);
    };

    const handleLogout = () => {
        base44.auth.logout('/');
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
                    <TabsTrigger value="privacy" className="rounded-lg text-xs gap-1.5"><Lock className="w-3.5 h-3.5" />Приватность</TabsTrigger>
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
                                <p className="text-xs text-muted-foreground pt-6">Нажми для изменения аватара и баннера</p>
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
                        <Button onClick={handleSave} disabled={saving} className="w-full nexus-gradient border-0 text-white rounded-xl h-10 font-bold shadow-nexus">
                            {saving ? <LoadingSpinner size="sm" /> : 'Сохранить'}
                        </Button>
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