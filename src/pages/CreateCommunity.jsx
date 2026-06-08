import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { ArrowLeft, Upload, Plus, X } from 'lucide-react';
import { useToast } from '@/components/ui/use-toast';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import { motion } from 'framer-motion';
import { validateFileSize, UPLOAD_LIMITS_MB, limitLabelForCategory } from '@/lib/validateFileSize';

const CATEGORIES = ['anime', 'gaming', 'fandoms', 'roleplay', 'art', 'music', 'books', 'movies', 'sports', 'technology', 'science', 'lifestyle', 'other'];

export default function CreateCommunity() {
    const { user } = useAuth();
    const navigate = useNavigate();
    const { toast } = useToast();

    const [name, setName] = useState('');
    const [description, setDescription] = useState('');
    const [category, setCategory] = useState('');
    const [isPrivate, setIsPrivate] = useState(false);
    const [isNsfw, setIsNsfw] = useState(false);
    const [tags, setTags] = useState([]);
    const [tagInput, setTagInput] = useState('');
    const [rules, setRules] = useState([{ title: '', description: '' }]);
    const [avatarUrl, setAvatarUrl] = useState('');
    const [bannerUrl, setBannerUrl] = useState('');
    const [uploadingAvatar, setUploadingAvatar] = useState(false);
    const [uploadingBanner, setUploadingBanner] = useState(false);
    const [submitting, setSubmitting] = useState(false);

    const handleAvatarUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setUploadingAvatar(true);
        try {
            validateFileSize(file, UPLOAD_LIMITS_MB['community/avatars']);
            const { file_url } = await nexusApi.integrations.Core.UploadFile({ file, category: 'community/avatars' });
            setAvatarUrl(file_url);
            toast({ title: '✅ Аватар загружен' });
        } catch (err) {
            toast({ title: err.message || 'Не удалось загрузить аватар', variant: 'destructive' });
        } finally {
            setUploadingAvatar(false);
            e.target.value = '';
        }
    };

    const handleBannerUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setUploadingBanner(true);
        try {
            validateFileSize(file, UPLOAD_LIMITS_MB['community/banners']);
            const { file_url } = await nexusApi.integrations.Core.UploadFile({ file, category: 'community/banners' });
            setBannerUrl(file_url);
            toast({ title: '✅ Баннер загружен' });
        } catch (err) {
            toast({ title: err.message || 'Не удалось загрузить баннер', variant: 'destructive' });
        } finally {
            setUploadingBanner(false);
            e.target.value = '';
        }
    };

    const addTag = () => {
        const t = tagInput.trim().toLowerCase();
        if (t && !tags.includes(t) && tags.length < 10) {
            setTags([...tags, t]);
            setTagInput('');
        }
    };

    const addRule = () => setRules([...rules, { title: '', description: '' }]);

    const handleSubmit = async () => {
        if (!name.trim()) { toast({ title: 'Введите название', variant: 'destructive' }); return; }
        setSubmitting(true);
        const slug = name.toLowerCase().replace(/[^a-zа-яё0-9]/gi, '-').replace(/-+/g, '-');

        const community = await nexusApi.entities.Community.create({
            name: name.trim(),
            slug,
            description: description.trim(),
            category,
            avatar_url: avatarUrl,
            banner_url: bannerUrl,
            owner_id: user.id,
            tags,
            rules: rules.filter(r => r.title.trim()),
            is_private: isPrivate,
            is_nsfw: isNsfw,
            member_count: 1,
            post_count: 0,
            activity_level: 'low',
        });

        await nexusApi.entities.CommunityMember.create({
            user_id: user.id,
            community_id: community.id,
            role: 'owner',
        });

        toast({ title: '🎉 Сообщество создано!' });
        navigate(`/community/${community.id}`);
        setSubmitting(false);
    };

    return (
        <div className="max-w-2xl mx-auto px-4 py-4">
            <div className="flex items-center gap-3 mb-5">
                <Button variant="ghost" size="icon" className="rounded-xl" onClick={() => navigate(-1)}>
                    <ArrowLeft className="w-5 h-5" />
                </Button>
                <h1 className="text-xl font-display font-black">Создать сообщество</h1>
            </div>

            <div className="space-y-4">
                {/* Banner upload */}
                <div className="nexus-card overflow-hidden">
                    <label className="relative block h-32 cursor-pointer group">
                        {bannerUrl ? (
                            <img src={bannerUrl} className="w-full h-full object-cover" alt="" />
                        ) : (
                            <div className="w-full h-full nexus-gradient opacity-30 flex items-center justify-center">
                                {uploadingBanner ? <LoadingSpinner size="sm" /> : (
                                    <div className="text-center">
                                        <Upload className="w-8 h-8 text-primary mx-auto mb-1" />
                                        <p className="text-xs text-muted-foreground">Загрузить баннер</p>
                                    </div>
                                )}
                            </div>
                        )}
                        <div className="absolute inset-0 bg-black/30 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                            <Upload className="w-6 h-6 text-white" />
                        </div>
                        <input type="file" accept="image/*" className="hidden" onChange={handleBannerUpload} />
                    </label>

                    <div className="p-4 flex items-center gap-4">
                        <label className="relative cursor-pointer">
                            <div className="w-16 h-16 rounded-2xl border-2 border-dashed border-border/50 overflow-hidden bg-muted flex items-center justify-center">
                                {avatarUrl ? (
                                    <img src={avatarUrl} className="w-full h-full object-cover" alt="" />
                                ) : uploadingAvatar ? <LoadingSpinner size="sm" /> : (
                                    <Upload className="w-5 h-5 text-muted-foreground" />
                                )}
                            </div>
                            <input type="file" accept="image/*" className="hidden" onChange={handleAvatarUpload} />
                        </label>
                        <div>
                            <p className="text-sm font-semibold">Аватар и баннер</p>
                            <p className="text-xs text-muted-foreground">
                                Аватар {limitLabelForCategory('community/avatars')}, баннер {limitLabelForCategory('community/banners')}
                            </p>
                        </div>
                    </div>
                </div>

                <div className="nexus-card p-4 space-y-4">
                    {/* Name */}
                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Название *</Label>
                        <Input value={name} onChange={e => setName(e.target.value)} placeholder="Название сообщества" className="rounded-xl border-border/50 h-10 text-sm" maxLength={50} />
                    </div>

                    {/* Description */}
                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Описание</Label>
                        <Textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="О чём это сообщество?" className="rounded-xl border-border/50 text-sm min-h-20 resize-none" />
                    </div>

                    {/* Category */}
                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Категория</Label>
                        <Select value={category} onValueChange={setCategory}>
                            <SelectTrigger className="rounded-xl border-border/50 h-10 text-sm">
                                <SelectValue placeholder="Выберите категорию" />
                            </SelectTrigger>
                            <SelectContent>
                                {CATEGORIES.map(c => <SelectItem key={c} value={c} className="capitalize">{c}</SelectItem>)}
                            </SelectContent>
                        </Select>
                    </div>

                    {/* Tags */}
                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Теги</Label>
                        <div className="flex gap-2 mb-2">
                            <Input value={tagInput} onChange={e => setTagInput(e.target.value)} onKeyDown={e => e.key === 'Enter' && (e.preventDefault(), addTag())} placeholder="Добавить тег..." className="rounded-xl border-border/50 h-9 text-sm flex-1" />
                            <Button variant="outline" size="sm" className="rounded-xl h-9" onClick={addTag}><Plus className="w-4 h-4" /></Button>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                            {tags.map(tag => (
                                <Badge key={tag} className="bg-primary/10 text-primary border-0 gap-1 cursor-pointer" onClick={() => setTags(tags.filter(t => t !== tag))}>
                                    #{tag} <X className="w-3 h-3" />
                                </Badge>
                            ))}
                        </div>
                    </div>

                    {/* Rules */}
                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Правила</Label>
                        <div className="space-y-2">
                            {rules.map((rule, i) => (
                                <div key={i} className="flex gap-2">
                                    <div className="w-6 h-6 nexus-gradient rounded-lg flex items-center justify-center text-white text-xs font-black flex-shrink-0 mt-2">{i + 1}</div>
                                    <div className="flex-1 space-y-1">
                                        <Input value={rule.title} onChange={e => { const r = [...rules]; r[i].title = e.target.value; setRules(r); }} placeholder="Название правила" className="rounded-xl border-border/50 h-8 text-xs" />
                                        <Input value={rule.description} onChange={e => { const r = [...rules]; r[i].description = e.target.value; setRules(r); }} placeholder="Описание (необязательно)" className="rounded-xl border-border/50 h-8 text-xs" />
                                    </div>
                                    {rules.length > 1 && (
                                        <Button variant="ghost" size="icon" className="h-8 w-8 rounded-xl mt-2" onClick={() => setRules(rules.filter((_, j) => j !== i))}>
                                            <X className="w-3.5 h-3.5" />
                                        </Button>
                                    )}
                                </div>
                            ))}
                        </div>
                        <Button variant="ghost" size="sm" className="mt-2 rounded-xl gap-1 text-xs" onClick={addRule}>
                            <Plus className="w-3.5 h-3.5" />Добавить правило
                        </Button>
                    </div>

                    {/* Toggles */}
                    <div className="space-y-3 pt-2 border-t border-border/50">
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-sm font-semibold">Приватное</p>
                                <p className="text-xs text-muted-foreground">Только по приглашению</p>
                            </div>
                            <Switch checked={isPrivate} onCheckedChange={setIsPrivate} />
                        </div>
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-sm font-semibold">NSFW контент</p>
                                <p className="text-xs text-muted-foreground">18+ материалы</p>
                            </div>
                            <Switch checked={isNsfw} onCheckedChange={setIsNsfw} />
                        </div>
                    </div>

                    <Button
                        onClick={handleSubmit}
                        disabled={submitting || !name.trim()}
                        className="w-full nexus-gradient border-0 text-white rounded-xl h-11 font-bold shadow-nexus text-sm"
                    >
                        {submitting ? <LoadingSpinner size="sm" /> : '🚀 Создать сообщество'}
                    </Button>
                </div>
            </div>
        </div>
    );
}