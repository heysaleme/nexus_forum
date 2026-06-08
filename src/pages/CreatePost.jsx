import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { motion, AnimatePresence } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';
import { Type, Image, Link as LinkIcon, BarChart2, BookOpen, X, Plus, Upload, ArrowLeft, Video } from 'lucide-react';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import { validateFileSize, UPLOAD_LIMITS_MB, limitLabelForCategory } from '@/lib/validateFileSize';

const POST_TYPES = [
    { id: 'text', label: 'Текст', icon: Type },
    { id: 'image', label: 'Фото', icon: Image },
    { id: 'video', label: 'Видео', icon: Video },
    { id: 'link', label: 'Ссылка', icon: LinkIcon },
    { id: 'poll', label: 'Опрос', icon: BarChart2 },
];

export default function CreatePost() {
    const { user, triggerAuthModal } = useAuth();
    const navigate = useNavigate();
    const { toast } = useToast();
    const [searchParams] = useSearchParams();
    const communityIdParam = searchParams.get('community');

    const [type, setType] = useState('text');
    const [title, setTitle] = useState('');
    const [content, setContent] = useState('');
    const [linkUrl, setLinkUrl] = useState('');
    const [selectedCommunity, setSelectedCommunity] = useState(communityIdParam || '');
    const [communities, setCommunities] = useState([]);
    const [tags, setTags] = useState([]);
    const [tagInput, setTagInput] = useState('');
    const [pollOptions, setPollOptions] = useState(['', '']);
    const [status, setStatus] = useState('published');
    const [uploading, setUploading] = useState(false);
    const [mediaUrls, setMediaUrls] = useState([]);
    const [submitting, setSubmitting] = useState(false);
    const [isNsfw, setIsNsfw] = useState(false);
    const [isSpoiler, setIsSpoiler] = useState(false);

    useEffect(() => {
        if (!user) {
            triggerAuthModal("Для создания публикации необходимо войти в аккаунт или зарегистрироваться.");
            navigate('/');
            return;
        }
        loadCommunities();
    }, [user]);

    const loadCommunities = async () => {
        const memberships = await nexusApi.entities.CommunityMember.filter({ user_id: user.id });
        if (memberships.length > 0) {
            const communityIds = memberships.map(m => m.community_id);
            const allCommunities = await nexusApi.entities.Community.list('-name', 50);
            setCommunities(allCommunities.filter(c => communityIds.includes(c.id)));
        } else {
            const allCommunities = await nexusApi.entities.Community.list('-name', 20);
            setCommunities(allCommunities);
        }
    };

    const addTag = () => {
        const t = tagInput.trim().toLowerCase().replace(/[^a-zа-яё0-9_]/gi, '');
        if (t && !tags.includes(t) && tags.length < 5) {
            setTags([...tags, t]);
            setTagInput('');
        }
    };

    const handleImageUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setUploading(true);
        try {
            validateFileSize(file, UPLOAD_LIMITS_MB['posts/images']);
            const { file_url } = await nexusApi.integrations.Core.UploadFile({ file, category: 'posts/images' });
            setMediaUrls(prev => [...prev, file_url]);
            toast({ title: '✅ Изображение загружено' });
        } catch (err) {
            toast({ title: err.message || 'Не удалось загрузить изображение', variant: 'destructive' });
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    };

    const handleVideoUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        if (!file.type.startsWith('video/')) {
            toast({ title: 'Выберите видеофайл', variant: 'destructive' });
            return;
        }
        setUploading(true);
        try {
            validateFileSize(file, UPLOAD_LIMITS_MB['posts/videos']);
            const { file_url } = await nexusApi.integrations.Core.UploadFile({ file, category: 'posts/videos' });
            setMediaUrls([file_url]);
            toast({ title: '✅ Видео загружено' });
        } catch (err) {
            toast({ title: err.message || 'Не удалось загрузить видео', variant: 'destructive' });
        } finally {
            setUploading(false);
            e.target.value = '';
        }
    };

    const handleSubmit = async (statusOverride) => {
        if (!title.trim()) { toast({ title: 'Введите заголовок', variant: 'destructive' }); return; }
        if (!selectedCommunity) { toast({ title: 'Выберите сообщество', variant: 'destructive' }); return; }

        if (type === 'video' && mediaUrls.length === 0) {
            toast({ title: 'Загрузите видео', variant: 'destructive' });
            return;
        }

        setSubmitting(true);
        const community = communities.find(c => c.id === selectedCommunity);
        const postData = {
            title: title.trim(),
            content: content.trim(),
            type,
            author_id: user.id,
            author_username: user.full_name || user.email,
            author_avatar: user.avatar_url,
            community_id: selectedCommunity,
            community_name: community?.name || '',
            community_avatar: community?.avatar_url,
            media_urls: mediaUrls,
            link_url: type === 'link' ? linkUrl : undefined,
            tags,
            status: statusOverride || status,
            poll_options: type === 'poll' ? pollOptions.filter(o => o.trim()).map(o => ({ text: o, votes: 0 })) : undefined,
            score: 0,
            upvotes: 0,
            downvotes: 0,
            views: 0,
            comment_count: 0,
            is_nsfw: isNsfw,
            is_spoiler: isSpoiler,
        };

        const post = await nexusApi.entities.Post.create(postData);
        toast({ title: statusOverride === 'draft' ? '📝 Черновик сохранён' : '✅ Опубликовано!' });
        navigate(statusOverride === 'draft' ? '/profile' : `/post/${post.id}`);
        setSubmitting(false);
    };

    return (
        <div className="max-w-2xl mx-auto px-4 py-4">
            {/* Header */}
            <div className="flex items-center gap-3 mb-5">
                <Button variant="ghost" size="icon" className="rounded-xl" onClick={() => navigate(-1)}>
                    <ArrowLeft className="w-5 h-5" />
                </Button>
                <h1 className="text-xl font-display font-black">Создать публикацию</h1>
            </div>

            {/* Type selector */}
            <div className="flex gap-2 mb-4 overflow-x-auto scrollbar-hide pb-1">
                {POST_TYPES.map(({ id, label, icon: Icon }) => (
                    <motion.button
                        key={id}
                        whileTap={{ scale: 0.95 }}
                        onClick={() => setType(id)}
                        className={`flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-semibold flex-shrink-0 transition-all ${type === id ? 'nexus-gradient text-white shadow-nexus' : 'bg-muted/60 text-muted-foreground hover:bg-muted'
                            }`}
                    >
                        <Icon className="w-3.5 h-3.5" />
                        {label}
                    </motion.button>
                ))}
            </div>

            <div className="nexus-card p-4 space-y-4">
                {/* Community selector */}
                <div>
                    <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Сообщество</Label>
                    <Select value={selectedCommunity} onValueChange={setSelectedCommunity}>
                        <SelectTrigger className="rounded-xl border-border/50 text-sm h-10">
                            <SelectValue placeholder="Выберите сообщество" />
                        </SelectTrigger>
                        <SelectContent>
                            {communities.map(c => (
                                <SelectItem key={c.id} value={c.id}>
                                    <div className="flex items-center gap-2">
                                        <img src={c.avatar_url || `https://api.dicebear.com/7.x/shapes/svg?seed=${c.name}`} className="w-5 h-5 rounded object-cover" alt="" />
                                        {c.name}
                                    </div>
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </div>

                {/* Title */}
                <div>
                    <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Заголовок</Label>
                    <Input
                        value={title}
                        onChange={e => setTitle(e.target.value)}
                        placeholder="О чём эта публикация?"
                        className="rounded-xl border-border/50 text-sm h-10"
                        maxLength={300}
                    />
                    <p className="text-xs text-muted-foreground mt-1 text-right">{title.length}/300</p>
                </div>

                {/* Content by type */}
                {type === 'text' ? (
                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Текст</Label>
                        <Textarea
                            value={content}
                            onChange={e => setContent(e.target.value)}
                            placeholder="Напиши что-нибудь интересное..."
                            className="rounded-xl border-border/50 text-sm min-h-32 resize-none"
                        />
                    </div>
                ) : type === 'link' ? (
                    <div className="space-y-3">
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Ссылка</Label>
                            <Input
                                value={linkUrl}
                                onChange={e => setLinkUrl(e.target.value)}
                                placeholder="https://..."
                                className="rounded-xl border-border/50 text-sm h-10"
                            />
                        </div>
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Описание (необязательно)</Label>
                            <Textarea
                                value={content}
                                onChange={e => setContent(e.target.value)}
                                placeholder="Расскажи, почему эта ссылка интересна..."
                                className="rounded-xl border-border/50 text-sm min-h-20 resize-none"
                            />
                        </div>
                    </div>
                ) : type === 'video' ? (
                    <div className="space-y-3">
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">
                                Видео ({limitLabelForCategory('posts/videos')})
                            </Label>
                            <label className="flex flex-col items-center justify-center border-2 border-dashed border-border/50 rounded-xl p-8 cursor-pointer hover:border-primary/50 hover:bg-primary/5 transition-all">
                                {uploading ? <LoadingSpinner size="sm" /> : (
                                    <>
                                        <Video className="w-8 h-8 text-muted-foreground mb-2" />
                                        <span className="text-sm text-muted-foreground">Нажми для загрузки видео</span>
                                    </>
                                )}
                                <input type="file" accept="video/*" className="hidden" onChange={handleVideoUpload} />
                            </label>
                            {mediaUrls.length > 0 && (
                                <div className="mt-2 rounded-xl overflow-hidden">
                                    <video src={mediaUrls[0]} controls className="w-full max-h-64 bg-black" />
                                </div>
                            )}
                        </div>
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Описание (необязательно)</Label>
                            <Textarea
                                value={content}
                                onChange={e => setContent(e.target.value)}
                                placeholder="Добавь описание к видео..."
                                className="rounded-xl border-border/50 text-sm min-h-20 resize-none"
                            />
                        </div>
                    </div>
                ) : type === 'image' ? (
                    <div className="space-y-3">
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">
                                Изображения ({limitLabelForCategory('posts/images')})
                            </Label>
                            <label className="flex flex-col items-center justify-center border-2 border-dashed border-border/50 rounded-xl p-8 cursor-pointer hover:border-primary/50 hover:bg-primary/5 transition-all">
                                {uploading ? <LoadingSpinner size="sm" /> : (
                                    <>
                                        <Upload className="w-8 h-8 text-muted-foreground mb-2" />
                                        <span className="text-sm text-muted-foreground">Нажми для загрузки</span>
                                    </>
                                )}
                                <input type="file" accept="image/*" className="hidden" onChange={handleImageUpload} />
                            </label>
                            {mediaUrls.length > 0 && (
                                <div className="flex gap-2 flex-wrap mt-2">
                                    {mediaUrls.map((url, i) => (
                                        <div key={i} className="relative">
                                            <img src={url} className="w-20 h-20 rounded-xl object-cover" alt="" />
                                            <button
                                                onClick={() => setMediaUrls(prev => prev.filter((_, j) => j !== i))}
                                                className="absolute -top-1 -right-1 w-5 h-5 bg-destructive rounded-full flex items-center justify-center"
                                            >
                                                <X className="w-3 h-3 text-white" />
                                            </button>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Подпись к изображению (необязательно)</Label>
                            <Textarea
                                value={content}
                                onChange={e => setContent(e.target.value)}
                                placeholder="Добавь описание или контекст к изображению..."
                                className="rounded-xl border-border/50 text-sm min-h-20 resize-none"
                            />
                        </div>
                    </div>
                ) : type === 'poll' ? (
                    <div className="space-y-3">
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Описание опроса (необязательно)</Label>
                            <Textarea
                                value={content}
                                onChange={e => setContent(e.target.value)}
                                placeholder="Поясни суть вопроса..."
                                className="rounded-xl border-border/50 text-sm min-h-16 resize-none"
                            />
                        </div>
                        <div>
                            <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Варианты ответа</Label>
                            <div className="space-y-2">
                                {pollOptions.map((opt, i) => (
                                    <div key={i} className="flex gap-2">
                                        <Input
                                            value={opt}
                                            onChange={e => { const o = [...pollOptions]; o[i] = e.target.value; setPollOptions(o); }}
                                            placeholder={`Вариант ${i + 1}`}
                                            className="rounded-xl border-border/50 text-sm h-9"
                                        />
                                        {i >= 2 && (
                                            <Button variant="ghost" size="icon" className="h-9 w-9 rounded-xl" onClick={() => setPollOptions(prev => prev.filter((_, j) => j !== i))}>
                                                <X className="w-4 h-4" />
                                            </Button>
                                        )}
                                    </div>
                                ))}
                                {pollOptions.length < 6 && (
                                    <Button variant="ghost" size="sm" className="rounded-xl gap-1.5 text-xs" onClick={() => setPollOptions([...pollOptions, ''])}>
                                        <Plus className="w-3.5 h-3.5" />
                                        Добавить вариант
                                    </Button>
                                )}
                            </div>
                        </div>
                    </div>
                ) : null}


                {/* Tags */}
                <div>
                    <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Теги (до 5)</Label>
                    <div className="flex gap-2">
                        <Input
                            value={tagInput}
                            onChange={e => setTagInput(e.target.value)}
                            onKeyDown={e => e.key === 'Enter' && (e.preventDefault(), addTag())}
                            placeholder="Добавить тег..."
                            className="rounded-xl border-border/50 text-sm h-9 flex-1"
                        />
                        <Button variant="outline" size="sm" className="rounded-xl h-9" onClick={addTag}>
                            <Plus className="w-4 h-4" />
                        </Button>
                    </div>
                    {tags.length > 0 && (
                        <div className="flex flex-wrap gap-1.5 mt-2">
                            {tags.map(tag => (
                                <Badge key={tag} className="bg-primary/10 text-primary border-0 gap-1 cursor-pointer" onClick={() => setTags(tags.filter(t => t !== tag))}>
                                    #{tag}
                                    <X className="w-3 h-3" />
                                </Badge>
                            ))}
                        </div>
                    )}
                </div>

                {/* NSFW & Spoiler Options */}
                <div className="flex gap-6 py-1">
                    <label className="flex items-center gap-2 cursor-pointer select-none">
                        <input
                            type="checkbox"
                            checked={isNsfw}
                            onChange={(e) => setIsNsfw(e.target.checked)}
                            className="rounded border-border/50 text-primary focus:ring-primary w-4 h-4 accent-primary"
                        />
                        <span className="text-sm font-semibold text-muted-foreground">NSFW (18+)</span>
                    </label>
                    <label className="flex items-center gap-2 cursor-pointer select-none">
                        <input
                            type="checkbox"
                            checked={isSpoiler}
                            onChange={(e) => setIsSpoiler(e.target.checked)}
                            className="rounded border-border/50 text-primary focus:ring-primary w-4 h-4 accent-primary"
                        />
                        <span className="text-sm font-semibold text-muted-foreground">Спойлер</span>
                    </label>
                </div>

                {/* Actions */}
                <div className="flex gap-2 pt-2">
                    <Button
                        variant="outline"
                        onClick={() => handleSubmit('draft')}
                        disabled={submitting}
                        className="rounded-xl flex-1 text-sm h-10"
                    >
                        Черновик
                    </Button>
                    <Button
                        onClick={() => handleSubmit('published')}
                        disabled={submitting || !title.trim() || !selectedCommunity}
                        className="nexus-gradient border-0 text-white rounded-xl flex-1 text-sm h-10 shadow-nexus font-bold"
                    >
                        {submitting ? <LoadingSpinner size="sm" /> : 'Опубликовать'}
                    </Button>
                </div>
            </div>
        </div>
    );
}