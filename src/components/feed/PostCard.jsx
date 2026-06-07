import { Link, useNavigate } from 'react-router-dom';
import { ArrowUp, ArrowDown, MessageCircle, Share2, Bookmark, Eye, Trash2, Pin } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { nexusApi } from '@/api/nexusApi';
import { useState } from 'react';
import { useToast } from '@/components/ui/use-toast';
import { useAuth } from '@/lib/AuthContext';

function timeAgoShort(date) {
    if (!date) return '';
    const seconds = Math.floor((new Date() - new Date(date)) / 1000);
    if (seconds < 60) return `${seconds}с`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}м`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}ч`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}д`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months}мес`;
    return `${Math.floor(months / 12)}г`;
}

export default function PostCard({ post, currentUser, onVote, onDeleteSuccess }) {
    const { toast } = useToast();
    const navigate = useNavigate();
    const { triggerAuthModal } = useAuth();
    const [userVote, setUserVote] = useState(null);
    const [score, setScore] = useState(post.score || 0);
    const [saved, setSaved] = useState(false);
    const [revealed, setRevealed] = useState(false);

    const handleVote = async (e, value) => {
        e.stopPropagation();
        if (!currentUser) {
            triggerAuthModal("Для голосования необходимо войти в аккаунт или зарегистрироваться.");
            return;
        }
        const newVote = userVote === value ? null : value;
        const delta = (newVote || 0) - (userVote || 0);
        // Update UI immediately, no reload
        setUserVote(newVote);
        setScore(prev => prev + delta);
        if (newVote !== null) {
            nexusApi.entities.Vote.create({ user_id: currentUser.id, target_id: post.id, target_type: 'post', value });
        }
    };

    const handleSave = async (e) => {
        e.stopPropagation();
        if (!currentUser) {
            triggerAuthModal("Для сохранения публикаций необходимо войти в аккаунт или зарегистрироваться.");
            return;
        }
        setSaved(s => !s);
        if (!saved) {
            nexusApi.entities.SavedPost.create({ user_id: currentUser.id, post_id: post.id, post_title: post.title, post_community: post.community_name });
            toast({ title: '📌 Сохранено!' });
        }
    };

    const handleShare = (e) => {
        e.stopPropagation();
        navigator.clipboard?.writeText(window.location.origin + `/post/${post.id}`);
        toast({ title: '🔗 Ссылка скопирована!' });
    };

    const handleDelete = async (e) => {
        e.stopPropagation();
        if (!window.confirm('Вы уверены, что хотите удалить этот пост?')) return;
        try {
            await nexusApi.entities.Post.delete(post.id);
            toast({ title: '🗑️ Пост удален' });
            if (onDeleteSuccess) {
                onDeleteSuccess(post.id);
            } else {
                window.location.reload();
            }
        } catch (err) {
            toast({ title: 'Ошибка при удалении', variant: 'destructive' });
        }
    };

    const ago = timeAgoShort(post.created_date);

    return (
        <div
            onClick={() => navigate(`/post/${post.id}`)}
            className="bg-card border-b border-border/40 cursor-pointer hover:bg-muted/20 transition-colors"
        >
            {/* Header */}
            <div className="flex items-center gap-2 px-5 pt-4 pb-2">
                {post.is_pinned && (
                    <Badge className="bg-green-600 text-white gap-1 py-0.5 rounded text-[10px] uppercase font-black hover:bg-green-600">
                        <Pin className="w-2.5 h-2.5 fill-white" />
                        Закреплено
                    </Badge>
                )}
                {/* Desktop */}
                <Link to={`/community/${post.community_id}`} onClick={e => e.stopPropagation()} className="hidden sm:flex items-center gap-1.5 group">
                    <img src={post.community_avatar || `https://api.dicebear.com/7.x/shapes/svg?seed=${post.community_name}`} className="w-5 h-5 rounded object-cover" alt="" />
                    <span className="text-xs font-bold text-primary group-hover:underline">{post.community_name}</span>
                </Link>
                <span className="hidden sm:block text-muted-foreground text-xs">·</span>
                <Link to={`/user/${post.author_id}`} onClick={e => e.stopPropagation()} className="hidden sm:flex items-center gap-1 group">
                    <img src={post.author_avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${post.author_username}`} className="w-4 h-4 rounded-full object-cover" alt="" />
                    <span className="text-xs text-muted-foreground group-hover:text-foreground">{post.author_username}</span>
                </Link>

                {/* Mobile: community + author text only */}
                <Link to={`/community/${post.community_id}`} onClick={e => e.stopPropagation()} className="sm:hidden flex items-center gap-1.5 group">
                    <img src={post.community_avatar || `https://api.dicebear.com/7.x/shapes/svg?seed=${post.community_name}`} className="w-5 h-5 rounded object-cover" alt="" />
                    <span className="text-xs font-bold text-primary group-hover:underline">{post.community_name}</span>
                </Link>
                <span className="sm:hidden text-muted-foreground text-xs">·</span>
                <span className="sm:hidden text-xs text-muted-foreground">{post.author_username}</span>

                <span className="text-muted-foreground text-xs ml-auto">{ago} назад</span>
            </div>

            {/* Title + content — both clickable via parent div */}
            <h3 className="px-5 text-[15px] font-bold text-foreground leading-snug line-clamp-2 mb-1.5 flex items-center gap-1.5 flex-wrap">
                {post.title}
                {post.is_nsfw && <Badge variant="destructive" className="text-[9px] uppercase px-1.5 py-0 rounded font-black">NSFW</Badge>}
                {post.is_spoiler && <Badge className="text-[9px] bg-yellow-600 text-white uppercase px-1.5 py-0 rounded font-black hover:bg-yellow-600">SPOILER</Badge>}
            </h3>

            {post.tags?.length > 0 && (
                <div className="flex flex-wrap gap-1.5 px-5 mb-2">
                    {post.tags.slice(0, 3).map(tag => (
                        <Badge key={tag} variant="secondary" className="text-[10px] rounded-full px-2 py-0 bg-primary/10 text-primary border-0">#{tag}</Badge>
                    ))}
                </div>
            )}

            {((post.is_nsfw || post.is_spoiler) && !revealed) ? (
                <div className="px-5 py-3.5 bg-muted/30 border border-dashed border-border/60 mx-5 my-2 rounded-xl flex flex-col items-center justify-center gap-2 text-center" onClick={e => e.stopPropagation()}>
                    <p className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                        {post.is_nsfw && '⚠️ NSFW (18+)'} {post.is_nsfw && post.is_spoiler && '·'} {post.is_spoiler && '🎬 Спойлер'}
                    </p>
                    <button
                        onClick={() => setRevealed(true)}
                        className="bg-primary text-primary-foreground text-[11px] font-bold px-2.5 py-1 rounded-lg hover:bg-primary/90 transition-all"
                    >
                        Показать контент
                    </button>
                </div>
            ) : (
                <>
                    {post.content && (
                        <p className="px-5 text-[13px] text-muted-foreground line-clamp-2 leading-relaxed mb-2.5">
                            {post.content}
                        </p>
                    )}

                    {/* Images — shown for any post with media_urls */}
                    {post.media_urls?.length > 0 && (
                        <div className={`mt-2 mx-5 overflow-hidden rounded-xl ${post.media_urls.length > 1 ? 'grid grid-cols-2 gap-0.5' : ''}`}>
                            {post.media_urls.slice(0, 4).map((url, i) => (
                                <div key={i} className="relative">
                                    <img src={url} alt="" className="w-full object-cover max-h-64" />
                                    {i === 3 && post.media_urls.length > 4 && (
                                        <div className="absolute inset-0 bg-black/50 flex items-center justify-center">
                                            <span className="text-white font-bold text-lg">+{post.media_urls.length - 4}</span>
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}

                    {/* Link preview in card */}
                    {post.type === 'link' && post.link_url && (
                        <a
                            href={post.link_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            onClick={e => e.stopPropagation()}
                            className="mx-5 mt-2 mb-1 flex items-center gap-2.5 bg-muted/40 border border-border/40 rounded-xl px-3 py-2 hover:bg-muted/60 transition-colors"
                        >
                            <div className="w-7 h-7 bg-primary/10 rounded-lg flex items-center justify-center flex-shrink-0">
                                <svg className="w-3.5 h-3.5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" /></svg>
                            </div>
                            <span className="text-xs text-muted-foreground truncate">{post.link_url}</span>
                        </a>
                    )}

                    {/* Poll preview in card */}
                    {post.type === 'poll' && post.poll_options?.length > 0 && (
                        <div className="px-5 mt-2 mb-1 space-y-1.5">
                            {post.poll_options.slice(0, 3).map((opt, i) => (
                                <div key={i} className="flex items-center gap-2 bg-muted/30 border border-border/40 rounded-lg px-3 py-1.5">
                                    <div className="w-3 h-3 rounded-full border-2 border-primary/50 flex-shrink-0" />
                                    <span className="text-xs font-medium">{opt.text}</span>
                                    <span className="text-xs text-muted-foreground ml-auto">{opt.votes || 0}</span>
                                </div>
                            ))}
                            {post.poll_options.length > 3 && (
                                <p className="text-xs text-muted-foreground pl-1">+{post.poll_options.length - 3} вариантов</p>
                            )}
                        </div>
                    )}
                </>
            )}


            {/* Actions */}
            <div className="flex items-center gap-1.5 px-4 py-3" onClick={e => e.stopPropagation()}>
                {/* Vote */}
                <div className="flex items-center bg-muted/50 rounded overflow-hidden">
                    <button
                        onClick={e => handleVote(e, 1)}
                        className={`h-7 px-2.5 flex items-center gap-1 text-xs font-bold transition-colors ${userVote === 1 ? 'text-primary bg-primary/10' : 'text-muted-foreground hover:text-foreground'}`}
                    >
                        <ArrowUp className="w-3.5 h-3.5" />
                        {score !== 0 ? score : ''}
                    </button>
                    <div className="w-px h-3.5 bg-border" />
                    <button
                        onClick={e => handleVote(e, -1)}
                        className={`h-7 px-2 flex items-center text-xs transition-colors ${userVote === -1 ? 'text-destructive bg-destructive/10' : 'text-muted-foreground hover:text-foreground'}`}
                    >
                        <ArrowDown className="w-3.5 h-3.5" />
                    </button>
                </div>

                <div className="flex items-center gap-1 h-7 px-2 text-xs text-muted-foreground">
                    <MessageCircle className="w-3.5 h-3.5" />
                    {post.comment_count || 0}
                </div>

                <div className="flex items-center gap-1 h-7 px-2 text-xs text-muted-foreground">
                    <Eye className="w-3 h-3" />
                    {post.views || 0}
                </div>

                <div className="ml-auto flex items-center gap-0.5">
                    {currentUser && (currentUser.id === post.author_id || currentUser.role === 'admin' || currentUser.role === 'moderator') && (
                        <button
                            onClick={handleDelete}
                            className="h-7 w-7 flex items-center justify-center text-destructive hover:text-destructive/80 transition-colors mr-1"
                            title="Удалить пост"
                        >
                            <Trash2 className="w-3.5 h-3.5" />
                        </button>
                    )}
                    <button
                        onClick={handleSave}
                        className={`h-7 w-7 flex items-center justify-center transition-colors ${saved ? 'text-primary' : 'text-muted-foreground hover:text-foreground'}`}
                    >
                        <Bookmark className={`w-3.5 h-3.5 ${saved ? 'fill-primary' : ''}`} />
                    </button>
                    <button
                        onClick={handleShare}
                        className="h-7 w-7 flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                    >
                        <Share2 className="w-3.5 h-3.5" />
                    </button>
                </div>
            </div>
        </div>
    );
}
