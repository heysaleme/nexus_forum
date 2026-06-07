import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import ReportModal from '@/components/ui/ReportModal';
import { ArrowUp, ArrowDown, MessageCircle, Share2, ArrowLeft, Send, Pin, Trash2, ExternalLink, BarChart2, Flag } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';

function timeAgoShort(date) {
    if (!date) return '';
    const seconds = Math.floor((new Date() - new Date(date)) / 1000);
    if (seconds < 60) return `${seconds}с назад`;
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}м назад`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}ч назад`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}д назад`;
    const months = Math.floor(days / 30);
    if (months < 12) return `${months}мес назад`;
    return `${Math.floor(months / 12)}г назад`;
}

const COMMENT_SORTS = [
    { id: 'top', label: 'Топ' },
    { id: 'new', label: 'Новые' },
    { id: 'old', label: 'Старые' },
];

// ── PollDisplay ──────────────────────────────────────────
function PollDisplay({ post, currentUser, onVote }) {
    const options = post.poll_options || [];
    const totalVotes = options.reduce((sum, o) => sum + (o.votes || 0), 0);
    return (
        <div className="px-4 pb-3 space-y-2">
            {options.map((opt, i) => {
                const pct = totalVotes > 0 ? Math.round(((opt.votes || 0) / totalVotes) * 100) : 0;
                return (
                    <button
                        key={i}
                        onClick={() => currentUser && onVote(i)}
                        className="w-full text-left"
                        disabled={!currentUser}
                    >
                        <div className="relative h-9 rounded-xl overflow-hidden border border-border/50 bg-muted/30 hover:border-primary/40 transition-colors">
                            <div
                                className="absolute inset-y-0 left-0 nexus-gradient opacity-20 transition-all"
                                style={{ width: `${pct}%` }}
                            />
                            <div className="relative flex items-center justify-between px-3 h-full">
                                <span className="text-sm font-semibold">{opt.text}</span>
                                <span className="text-xs text-muted-foreground font-bold">{pct}%</span>
                            </div>
                        </div>
                    </button>
                );
            })}
            <p className="text-xs text-muted-foreground">{totalVotes} голосов</p>
        </div>
    );
}

// ── LinkPreview ──────────────────────────────────────────
function LinkPreview({ url }) {
    if (!url) return null;
    let display = url;
    try { display = new URL(url).hostname; } catch {}
    return (
        <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="mx-4 mb-3 flex items-center gap-3 bg-muted/40 border border-border/50 rounded-xl p-3 hover:bg-muted/60 hover:border-primary/30 transition-all group"
        >
            <div className="w-10 h-10 bg-primary/10 rounded-lg flex items-center justify-center flex-shrink-0">
                <ExternalLink className="w-4 h-4 text-primary" />
            </div>
            <div className="min-w-0">
                <p className="text-xs text-muted-foreground truncate">{display}</p>
                <p className="text-sm font-semibold truncate group-hover:text-primary transition-colors">{url}</p>
            </div>
            <ExternalLink className="w-4 h-4 text-muted-foreground ml-auto flex-shrink-0" />
        </a>
    );
}

// ── CommentItem ──────────────────────────────────────────
function CommentItem({ comment, depth = 0, currentUser, postId, onReload, canDeleteOthers }) {
    const { toast } = useToast();
    const { triggerAuthModal } = useAuth();
    const [showReply, setShowReply] = useState(false);
    const [replyText, setReplyText] = useState('');
    const [userVote, setUserVote] = useState(null);
    const [score, setScore] = useState(comment.score || 0);
    const [isEditing, setIsEditing] = useState(false);
    const [editText, setEditText] = useState(comment.content || '');
    const [reportOpen, setReportOpen] = useState(false);

    const handleVote = (value) => {
        if (!currentUser) {
            triggerAuthModal("Для голосования необходимо войти в аккаунт или зарегистрироваться.");
            return;
        }
        const newVote = userVote === value ? null : value;
        setUserVote(newVote);
        setScore(prev => prev + ((newVote || 0) - (userVote || 0)));
        if (newVote !== null) {
            nexusApi.entities.Vote.create({ user_id: currentUser.id, target_id: comment.id, target_type: 'comment', value });
        }
    };

    const handleReply = async () => {
        if (!replyText.trim()) return;
        await nexusApi.entities.Comment.create({
            post_id: postId,
            author_id: currentUser.id,
            author_username: currentUser.full_name || currentUser.email,
            author_avatar: currentUser.avatar_url,
            parent_id: comment.id,
            content: replyText.trim(),
            depth: depth + 1,
            score: 0,
        });
        setReplyText('');
        setShowReply(false);
        onReload();
    };

    const handleSaveEdit = async () => {
        if (!editText.trim()) return;
        try {
            await nexusApi.entities.Comment.update(comment.id, { content: editText.trim() });
            toast({ title: '✨ Комментарий обновлен' });
            setIsEditing(false);
            onReload();
        } catch (err) {
            toast({ title: 'Не удалось обновить комментарий', variant: 'destructive' });
        }
    };

    const handleDeleteComment = async () => {
        if (!window.confirm('Вы уверены, что хотите удалить этот комментарий?')) return;
        try {
            await nexusApi.entities.Comment.delete(comment.id);
            toast({ title: '🗑️ Комментарий удален' });
            onReload();
        } catch (err) {
            toast({ title: 'Не удалось удалить комментарий', variant: 'destructive' });
        }
    };

    return (
        <div className={`${depth > 0 ? 'ml-4 border-l-2 border-border/30 pl-3' : ''}`}>
            <div className="py-2.5">
                <div className="flex items-center gap-2 mb-1">
                    <img src={comment.author_avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${comment.author_username}`} className="w-6 h-6 rounded-full object-cover" alt="" />
                    <Link to={`/user/${comment.author_id}`} className="text-xs font-bold hover:text-primary">{comment.author_username}</Link>
                    {comment.author_level && <Badge className="text-[9px] h-4 px-1.5 bg-primary/10 text-primary border-0">Ур. {comment.author_level}</Badge>}
                    <span className="text-[10px] text-muted-foreground ml-auto">{timeAgoShort(comment.created_date)}</span>
                </div>

                {isEditing ? (
                    <div className="mt-2 ml-8 flex gap-2">
                        <Textarea value={editText} onChange={e => setEditText(e.target.value)} className="text-sm min-h-0 h-14 resize-none border-border/50 text-xs rounded" />
                        <div className="flex flex-col gap-1">
                            <Button size="sm" className="nexus-gradient border-0 text-white h-7 px-3 shadow-nexus rounded text-xs" onClick={handleSaveEdit}>Сохранить</Button>
                            <Button variant="ghost" size="sm" className="h-7 px-3 text-xs rounded" onClick={() => setIsEditing(false)}>Отмена</Button>
                        </div>
                    </div>
                ) : comment.is_deleted ? (
                    <p className="text-xs text-muted-foreground italic ml-8">{comment.content}</p>
                ) : (
                    <p className="text-sm text-foreground leading-relaxed ml-8">{comment.content}</p>
                )}

                <div className="flex items-center gap-2 mt-1.5 ml-8 flex-wrap">
                    <div className="flex items-center bg-muted/40 rounded overflow-hidden">
                        <button onClick={() => handleVote(1)} className={`px-2 py-0.5 text-xs font-bold flex items-center gap-1 ${userVote === 1 ? 'text-primary bg-primary/10' : 'text-muted-foreground hover:text-foreground'}`}>
                            <ArrowUp className="w-3 h-3" />{score > 0 ? score : ''}
                        </button>
                        <span className="w-px h-3 bg-border" />
                        <button onClick={() => handleVote(-1)} className={`px-2 py-0.5 text-xs ${userVote === -1 ? 'text-destructive bg-destructive/10' : 'text-muted-foreground hover:text-foreground'}`}>
                            <ArrowDown className="w-3 h-3" />
                        </button>
                    </div>
                    {currentUser && !comment.is_deleted && depth < 3 && (
                        <button onClick={() => setShowReply(!showReply)} className="text-xs text-muted-foreground hover:text-primary font-semibold">
                            Ответить
                        </button>
                    )}
                    {currentUser && !comment.is_deleted && currentUser.id === comment.author_id && (
                        <button onClick={() => { setEditText(comment.content); setIsEditing(true); }} className="text-xs text-muted-foreground hover:text-primary font-semibold ml-2">
                            Изменить
                        </button>
                    )}
                    {currentUser && !comment.is_deleted && (currentUser.id === comment.author_id || canDeleteOthers) && (
                        <button onClick={handleDeleteComment} className="text-xs text-destructive hover:text-destructive/80 font-semibold ml-2">
                            Удалить
                        </button>
                    )}
                    {/* Report button for all logged-in users who are not the author */}
                    {currentUser && !comment.is_deleted && currentUser.id !== comment.author_id && (
                        <button onClick={() => setReportOpen(true)} className="text-xs text-muted-foreground hover:text-orange-500 font-semibold ml-2 flex items-center gap-0.5">
                            <Flag className="w-3 h-3" /> Пожаловаться
                        </button>
                    )}
                </div>

                <AnimatePresence>
                    {showReply && (
                        <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: 'auto' }} exit={{ opacity: 0, height: 0 }} className="mt-2 ml-8 flex gap-2">
                            <Textarea value={replyText} onChange={e => setReplyText(e.target.value)} placeholder="Ответить..." className="text-sm min-h-0 h-14 resize-none border-border/50 text-xs rounded" />
                            <div className="flex flex-col gap-1">
                                <Button size="sm" className="nexus-gradient border-0 text-white h-7 w-7 p-0 shadow-nexus rounded" onClick={handleReply}><Send className="w-3 h-3" /></Button>
                                <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-xs rounded" onClick={() => setShowReply(false)}>✕</Button>
                            </div>
                        </motion.div>
                    )}
                </AnimatePresence>
            </div>

            {reportOpen && (
                <ReportModal
                    open={reportOpen}
                    onClose={() => setReportOpen(false)}
                    targetId={comment.id}
                    targetType="comment"
                    currentUser={currentUser}
                />
            )}
        </div>
    );
}

// ── PostPage ─────────────────────────────────────────────
export default function PostPage() {
    const { id } = useParams();
    const { user, triggerAuthModal } = useAuth();

    const navigate = useNavigate();
    const { toast } = useToast();
    const [post, setPost] = useState(null);
    const [comments, setComments] = useState([]);
    const [loading, setLoading] = useState(true);
    const [userVote, setUserVote] = useState(null);
    const [score, setScore] = useState(0);
    const [newComment, setNewComment] = useState('');
    const [submittingComment, setSubmittingComment] = useState(false);
    const [commentSort, setCommentSort] = useState('top');
    const [isEditing, setIsEditing] = useState(false);
    const [editTitle, setEditTitle] = useState('');
    const [editContent, setEditContent] = useState('');
    const [editMediaUrls, setEditMediaUrls] = useState([]);
    const [editUploading, setEditUploading] = useState(false);
    const [revealed, setRevealed] = useState(false);
    const [memberRole, setMemberRole] = useState(null);
    const [reportOpen, setReportOpen] = useState(false);

    useEffect(() => { loadPost(); }, [id, user]);

    const loadPost = async () => {
        setLoading(true);
        const [posts, commentsData] = await Promise.all([
            nexusApi.entities.Post.filter({ id }),
            nexusApi.entities.Comment.filter({ post_id: id }, 'created_date'),
        ]);
        if (posts[0]) {
            setPost(posts[0]);
            setScore(posts[0].score || 0);
            nexusApi.entities.Post.update(posts[0].id, { views: (posts[0].views || 0) + 1 });

            if (user) {
                const mems = await nexusApi.entities.CommunityMember.filter({ user_id: user.id, community_id: posts[0].community_id }).catch(() => []);
                if (mems[0]) {
                    setMemberRole(mems[0].role);
                } else {
                    setMemberRole(null);
                }
            }
        }
        setComments(commentsData);
        setLoading(false);
    };

    const handleVote = (value) => {
        if (!user) {
            triggerAuthModal("Для голосования необходимо войти в аккаунт или зарегистрироваться.");
            return;
        }
        const newVote = userVote === value ? null : value;
        const delta = (newVote || 0) - (userVote || 0);
        setUserVote(newVote);
        setScore(prev => prev + delta);
        if (newVote !== null) {
            nexusApi.entities.Vote.create({ user_id: user.id, target_id: post.id, target_type: 'post', value });
        }
        nexusApi.entities.Post.update(post.id, { score: score + delta });
    };

    const handlePollVote = async (optionIndex) => {
        if (!user) {
            triggerAuthModal("Войдите, чтобы проголосовать.");
            return;
        }
        const opts = [...(post.poll_options || [])];
        opts[optionIndex] = { ...opts[optionIndex], votes: (opts[optionIndex].votes || 0) + 1 };
        await nexusApi.entities.Post.update(post.id, { poll_options: opts });
        setPost(prev => ({ ...prev, poll_options: opts }));
        toast({ title: '✅ Голос учтен!' });
    };

    const isAuthor = user && post && user.id === post.author_id;
    const isGlobalAdminOrMod = user && (user.role === 'admin' || user.role === 'moderator');
    const isCommOwnerOrMod = memberRole === 'owner' || memberRole === 'moderator';
    const canDelete = isAuthor || isGlobalAdminOrMod || isCommOwnerOrMod;
    const canPin = isGlobalAdminOrMod || memberRole === 'owner';
    const canEdit = isAuthor;

    const handleDeletePost = async () => {
        if (!window.confirm('Вы уверены, что хотите удалить этот пост?')) return;
        try {
            await nexusApi.entities.Post.delete(post.id);
            toast({ title: '🗑️ Пост удален' });
            navigate('/');
        } catch (err) {
            toast({ title: 'Не удалось удалить пост', variant: 'destructive' });
        }
    };

    const handleTogglePin = async () => {
        try {
            const newPinned = !post.is_pinned;
            await nexusApi.entities.Post.update(post.id, { is_pinned: newPinned });
            setPost(prev => ({ ...prev, is_pinned: newPinned }));
            toast({ title: newPinned ? '📌 Пост закреплен' : '📌 Пост откреплен' });
        } catch (err) {
            toast({ title: 'Не удалось обновить закрепление', variant: 'destructive' });
        }
    };

    const startEditing = () => {
        setEditTitle(post.title);
        setEditContent(post.content || '');
        setEditMediaUrls(post.media_urls || []);
        setIsEditing(true);
    };

    const handleEditImageUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;
        setEditUploading(true);
        const { file_url } = await nexusApi.integrations.Core.UploadFile({ file });
        setEditMediaUrls(prev => [...prev, file_url]);
        setEditUploading(false);
    };

    const handleSaveEdit = async () => {
        if (!editTitle.trim()) {
            toast({ title: 'Заголовок не может быть пустым', variant: 'destructive' });
            return;
        }
        try {
            const updated = await nexusApi.entities.Post.update(post.id, {
                title: editTitle.trim(),
                content: editContent.trim(),
                media_urls: editMediaUrls,
            });
            setPost(prev => ({ ...prev, title: updated.title, content: updated.content, media_urls: editMediaUrls }));
            setIsEditing(false);
            toast({ title: '✨ Публикация обновлена' });
        } catch (err) {
            toast({ title: 'Не удалось обновить публикацию', variant: 'destructive' });
        }
    };

    const handleComment = async () => {
        if (!newComment.trim() || !user) return;
        setSubmittingComment(true);
        await nexusApi.entities.Comment.create({
            post_id: id,
            author_id: user.id,
            author_username: user.full_name || user.email,
            author_avatar: user.avatar_url,
            content: newComment.trim(),
            score: 0,
            depth: 0,
        });
        await nexusApi.entities.Post.update(id, { comment_count: (post.comment_count || 0) + 1 });
        setNewComment('');
        setSubmittingComment(false);
        loadPost();
    };

    const buildCommentTree = (allComments, parentId = null) =>
        allComments.filter(c => (c.parent_id || null) === parentId).map(c => ({ ...c, children: buildCommentTree(allComments, c.id) }));

    const sortedComments = [...comments].sort((a, b) => {
        if (commentSort === 'top') return (b.score || 0) - (a.score || 0);
        if (commentSort === 'new') return new Date(b.created_date) - new Date(a.created_date);
        return new Date(a.created_date) - new Date(b.created_date);
    });

    const renderComments = (list) => list.map(comment => (
        <div key={comment.id}>
            <CommentItem
                comment={comment}
                depth={comment.depth || 0}
                currentUser={user}
                postId={id}
                onReload={loadPost}
                canDeleteOthers={isGlobalAdminOrMod || memberRole === 'owner' || memberRole === 'moderator'}
            />
            {comment.children?.length > 0 && <div>{renderComments(comment.children)}</div>}
            <div className="border-b border-border/20 last:border-0" />
        </div>
    ));

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;
    if (!post) return <EmptyState icon={MessageCircle} title="Публикация не найдена" />;

    const topLevelComments = buildCommentTree(sortedComments);
    // Content is blocked by spoiler/nsfw only when not yet revealed
    const contentBlocked = (post.is_nsfw || post.is_spoiler) && !revealed;

    return (
        <div className="max-w-3xl mx-auto">
            <div className="px-4 pt-3">
                <Button variant="ghost" size="sm" className="rounded gap-2 mb-3 text-sm -ml-2" onClick={() => navigate(-1)}>
                    <ArrowLeft className="w-4 h-4" />Назад
                </Button>
            </div>

            {/* Single white container for everything */}
            <div className="bg-card border-y border-border/40">

                {/* Post header */}
                <div className="flex items-center gap-2 px-4 pt-3 pb-1.5 flex-wrap">
                    <Link to={`/community/${post.community_id}`} className="flex items-center gap-1.5 group">
                        <img src={post.community_avatar || `https://api.dicebear.com/7.x/shapes/svg?seed=${post.community_name}`} className="w-5 h-5 rounded object-cover" alt="" />
                        <span className="text-xs font-bold text-primary group-hover:underline">{post.community_name}</span>
                    </Link>
                    <span className="text-muted-foreground text-xs">·</span>
                    <Link to={`/user/${post.author_id}`} className="flex items-center gap-1.5 group">
                        <img src={post.author_avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${post.author_username}`} className="w-4 h-4 rounded-full" alt="" />
                        <span className="text-xs text-muted-foreground group-hover:text-foreground">{post.author_username}</span>
                    </Link>
                    <span className="text-muted-foreground text-xs">·</span>
                    <span className="text-xs text-muted-foreground">{timeAgoShort(post.created_date)}</span>

                    {/* Moderation / Edit Actions */}
                    <div className="ml-auto flex items-center gap-1.5 flex-wrap">
                        {post.is_pinned && (
                            <Badge className="bg-green-600 text-white gap-1 py-0.5 rounded text-[10px] uppercase font-black hover:bg-green-600">
                                <Pin className="w-2.5 h-2.5 fill-white" />
                                Закреплено
                            </Badge>
                        )}
                        {canPin && (
                            <Button variant="ghost" size="sm" onClick={handleTogglePin} className="h-7 px-2 text-xs rounded-lg text-primary hover:bg-primary/10">
                                <Pin className="w-3.5 h-3.5 mr-1" />
                                {post.is_pinned ? 'Открепить' : 'Закрепить'}
                            </Button>
                        )}
                        {canEdit && !isEditing && (
                            <Button variant="ghost" size="sm" onClick={startEditing} className="h-7 px-2 text-xs rounded-lg text-foreground hover:bg-muted">
                                Редактировать
                            </Button>
                        )}
                        {canDelete && (
                            <Button variant="ghost" size="sm" onClick={handleDeletePost} className="h-7 px-2 text-xs rounded-lg text-destructive hover:bg-destructive/10">
                                Удалить
                            </Button>
                        )}
                        {/* Report button for non-authors */}
                        {user && !isAuthor && (
                            <Button variant="ghost" size="sm" onClick={() => setReportOpen(true)} className="h-7 px-2 text-xs rounded-lg text-muted-foreground hover:text-orange-500 hover:bg-orange-50 gap-1">
                                <Flag className="w-3 h-3" /> Жалоба
                            </Button>
                        )}
                    </div>
                </div>

                {isEditing ? (
                    <div className="px-4 py-3 space-y-3">
                        <input
                            type="text"
                            value={editTitle}
                            onChange={e => setEditTitle(e.target.value)}
                            className="w-full text-base font-bold bg-muted/50 border border-border/50 rounded-xl px-3 py-2 text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                            placeholder="Заголовок"
                        />
                        <textarea
                            value={editContent}
                            onChange={e => setEditContent(e.target.value)}
                            className="w-full min-h-24 text-sm bg-muted/50 border border-border/50 rounded-xl px-3 py-2 text-foreground focus:outline-none focus:ring-1 focus:ring-primary resize-none"
                            placeholder="Текст публикации..."
                        />
                        {/* Image management */}
                        {(post.type === 'image' || editMediaUrls.length > 0) && (
                            <div>
                                <p className="text-xs font-semibold text-muted-foreground mb-2">Изображения</p>
                                {editMediaUrls.length > 0 && (
                                    <div className="flex gap-2 flex-wrap mb-2">
                                        {editMediaUrls.map((url, i) => (
                                            <div key={i} className="relative">
                                                <img src={url} className="w-24 h-24 rounded-xl object-cover" alt="" />
                                                <button
                                                    onClick={() => setEditMediaUrls(prev => prev.filter((_, j) => j !== i))}
                                                    className="absolute -top-1.5 -right-1.5 w-5 h-5 bg-destructive rounded-full flex items-center justify-center text-white text-xs"
                                                >✕</button>
                                            </div>
                                        ))}
                                    </div>
                                )}
                                <label className="flex items-center gap-2 cursor-pointer text-xs text-primary font-semibold hover:underline">
                                    {editUploading ? <LoadingSpinner size="sm" /> : '+ Добавить изображение'}
                                    <input type="file" accept="image/*" className="hidden" onChange={handleEditImageUpload} />
                                </label>
                            </div>
                        )}
                        <div className="flex gap-2">
                            <Button size="sm" onClick={handleSaveEdit} className="nexus-gradient border-0 text-white rounded-lg shadow-nexus text-xs h-8 px-3">
                                Сохранить
                            </Button>
                            <Button variant="ghost" size="sm" onClick={() => setIsEditing(false)} className="rounded-lg text-xs h-8 px-3">
                                Отмена
                            </Button>
                        </div>
                    </div>
                ) : (
                    <>
                        <h1 className="px-4 text-xl font-display font-black text-foreground leading-snug mb-2 flex items-center gap-2 flex-wrap">
                            {post.title}
                            {post.is_nsfw && <Badge variant="destructive" className="text-[9px] uppercase px-1.5 py-0 rounded font-black">NSFW</Badge>}
                            {post.is_spoiler && <Badge className="text-[9px] bg-yellow-600 text-white uppercase px-1.5 py-0 rounded font-black hover:bg-yellow-600">SPOILER</Badge>}
                            {post.type === 'poll' && <Badge className="text-[9px] bg-blue-100 text-blue-700 border-0 gap-1 rounded font-bold"><BarChart2 className="w-2.5 h-2.5" />Опрос</Badge>}
                        </h1>

                        {post.tags?.length > 0 && (
                            <div className="flex flex-wrap gap-1.5 px-4 mb-2">
                                {post.tags.map(tag => <Badge key={tag} className="bg-primary/10 text-primary border-0 text-xs rounded-full">#{tag}</Badge>)}
                            </div>
                        )}

                        {contentBlocked ? (
                            <div className="px-4 py-6 bg-muted/30 border border-dashed border-border/60 mx-4 my-2 rounded-xl flex flex-col items-center justify-center gap-2 text-center">
                                <p className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                                    {post.is_nsfw && '⚠️ NSFW (18+)'} {post.is_nsfw && post.is_spoiler && '·'} {post.is_spoiler && '🎬 Спойлер'}
                                </p>
                                <button
                                    onClick={() => setRevealed(true)}
                                    className="bg-primary text-primary-foreground text-xs font-bold px-3 py-1.5 rounded-lg hover:bg-primary/90 transition-all"
                                >
                                    Показать контент
                                </button>
                            </div>
                        ) : (
                            <>
                                {/* Text content */}
                                {post.content && <p className="px-4 text-sm text-foreground leading-relaxed mb-3 whitespace-pre-wrap">{post.content}</p>}

                                {/* Images — shown for image type or any post with media */}
                                {post.media_urls?.length > 0 && (
                                    <div className={`mx-4 mb-3 overflow-hidden rounded-xl ${post.media_urls.length > 1 ? 'grid grid-cols-2 gap-0.5' : ''}`}>
                                        {post.media_urls.map((url, i) => (
                                            <img key={i} src={url} className="w-full object-cover max-h-[500px]" alt="" />
                                        ))}
                                    </div>
                                )}

                                {/* Link preview */}
                                {post.type === 'link' && post.link_url && <LinkPreview url={post.link_url} />}

                                {/* Poll */}
                                {post.type === 'poll' && post.poll_options?.length > 0 && (
                                    <PollDisplay post={post} currentUser={user} onVote={handlePollVote} />
                                )}
                            </>
                        )}
                    </>
                )}

                {/* Post vote actions */}
                <div className="flex items-center gap-1 px-3 pb-3">
                    <div className="flex items-center bg-muted/60 rounded overflow-hidden">
                        <button onClick={() => handleVote(1)} className={`h-8 px-3 flex items-center gap-1.5 font-bold text-sm ${userVote === 1 ? 'text-primary bg-primary/10' : 'text-muted-foreground'}`}>
                            <ArrowUp className="w-4 h-4" />{score}
                        </button>
                        <div className="w-px h-5 bg-border" />
                        <button onClick={() => handleVote(-1)} className={`h-8 px-3 flex items-center ${userVote === -1 ? 'text-destructive bg-destructive/10' : 'text-muted-foreground'}`}>
                            <ArrowDown className="w-4 h-4" />
                        </button>
                    </div>
                    <div className="flex items-center gap-1.5 h-8 px-2 text-sm text-muted-foreground">
                        <MessageCircle className="w-4 h-4" />{comments.length}
                    </div>
                    <button onClick={() => { navigator.clipboard?.writeText(window.location.href); toast({ title: '🔗 Ссылка скопирована' }); }} className="h-8 px-3 text-muted-foreground hover:text-foreground">
                        <Share2 className="w-4 h-4" />
                    </button>
                </div>

                {/* Divider */}
                <div className="border-t border-border/40" />

                {/* Comments header + sort */}
                <div className="px-4 py-2.5 flex items-center justify-between">
                    <h3 className="font-bold text-sm flex items-center gap-1.5">
                        <MessageCircle className="w-4 h-4 text-primary" />
                        Комментарии ({comments.length})
                    </h3>
                    <div className="flex gap-1">
                        {COMMENT_SORTS.map(s => (
                            <button key={s.id} onClick={() => setCommentSort(s.id)} className={`px-2.5 py-1 text-xs font-semibold rounded transition-colors ${commentSort === s.id ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:text-foreground'}`}>
                                {s.label}
                            </button>
                        ))}
                    </div>
                </div>

                {/* Divider */}
                <div className="border-t border-border/40" />

                {/* Comment input */}
                {user ? (
                    <div className="px-4 py-2.5 border-b border-border/40">
                        <div className="flex gap-2.5 items-center">
                            <img src={user.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${user.email}`} className="w-7 h-7 rounded-full flex-shrink-0" alt="" />
                            <div className="flex-1 flex gap-2">
                                <Textarea
                                    value={newComment}
                                    onChange={e => setNewComment(e.target.value)}
                                    onKeyDown={e => e.key === 'Enter' && e.ctrlKey && handleComment()}
                                    placeholder="Написать комментарий..."
                                    className="text-sm min-h-0 h-9 resize-none flex-1 py-1.5 rounded border-border/50"
                                    style={{ minHeight: '36px' }}
                                />
                                <Button onClick={handleComment} disabled={!newComment.trim() || submittingComment} size="sm" className="nexus-gradient border-0 text-white rounded shadow-nexus h-9 w-9 p-0 flex-shrink-0">
                                    <Send className="w-3.5 h-3.5" />
                                </Button>
                            </div>
                        </div>
                    </div>
                ) : (
                    <div className="px-4 py-2.5 border-b border-border/40 flex items-center justify-between">
                        <p className="text-xs text-muted-foreground">Войдите, чтобы оставить комментарий</p>
                        <Link to="/login"><Button size="sm" className="nexus-gradient border-0 text-white rounded shadow-nexus h-7 px-3 text-xs">Войти</Button></Link>
                    </div>
                )}

                {/* Comments list */}
                {comments.length === 0 ? (
                    <EmptyState icon={MessageCircle} title="Комментариев пока нет" description="Будь первым!" />
                ) : (
                    <div className="divide-y divide-border/20 px-4">
                        {renderComments(topLevelComments)}
                    </div>
                )}

            </div>

            {/* Report modal for post */}
            {reportOpen && (
                <ReportModal
                    open={reportOpen}
                    onClose={() => setReportOpen(false)}
                    targetId={post.id}
                    targetType="post"
                    currentUser={user}
                />
            )}
        </div>
    );
}