import { Link, useNavigate } from 'react-router-dom';
import { ArrowUp, ArrowDown, MessageCircle, Share2, Bookmark, Eye } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { base44 } from '@/api/base44Client';
import { useState } from 'react';
import { useToast } from '@/components/ui/use-toast';

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

export default function PostCard({ post, currentUser, onVote }) {
    const { toast } = useToast();
    const navigate = useNavigate();
    const [userVote, setUserVote] = useState(null);
    const [score, setScore] = useState(post.score || 0);
    const [saved, setSaved] = useState(false);

    const handleVote = async (e, value) => {
        e.stopPropagation();
        if (!currentUser) {
            toast({ title: 'Войдите, чтобы голосовать', variant: 'destructive' });
            return;
        }
        const newVote = userVote === value ? null : value;
        const delta = (newVote || 0) - (userVote || 0);
        // Update UI immediately, no reload
        setUserVote(newVote);
        setScore(prev => prev + delta);
        if (newVote !== null) {
            base44.entities.Vote.create({ user_id: currentUser.id, target_id: post.id, target_type: 'post', value });
        }
    };

    const handleSave = async (e) => {
        e.stopPropagation();
        if (!currentUser) return;
        setSaved(s => !s);
        if (!saved) {
            base44.entities.SavedPost.create({ user_id: currentUser.id, post_id: post.id, post_title: post.title, post_community: post.community_name });
            toast({ title: '📌 Сохранено!' });
        }
    };

    const handleShare = (e) => {
        e.stopPropagation();
        navigator.clipboard?.writeText(window.location.origin + `/post/${post.id}`);
        toast({ title: '🔗 Ссылка скопирована!' });
    };

    const ago = timeAgoShort(post.created_date);

    return (
        <div
            onClick={() => navigate(`/post/${post.id}`)}
            className="bg-card border-b border-border/40 cursor-pointer hover:bg-muted/20 transition-colors"
        >
            {/* Header */}
            <div className="flex items-center gap-2 px-5 pt-4 pb-2">
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
            <h3 className="px-5 text-[15px] font-bold text-foreground leading-snug line-clamp-2 mb-1.5">
                {post.title}
            </h3>

            {post.tags?.length > 0 && (
                <div className="flex flex-wrap gap-1.5 px-5 mb-2">
                    {post.tags.slice(0, 3).map(tag => (
                        <Badge key={tag} variant="secondary" className="text-[10px] rounded-full px-2 py-0 bg-primary/10 text-primary border-0">#{tag}</Badge>
                    ))}
                </div>
            )}

            {post.content && (
                <p className="px-5 text-[13px] text-muted-foreground line-clamp-2 leading-relaxed mb-2.5">
                    {post.content}
                </p>
            )}

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
