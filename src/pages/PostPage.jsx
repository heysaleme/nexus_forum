import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { base44 } from '@/api/base44Client';
import { useAuth } from '@/lib/AuthContext';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { ArrowUp, ArrowDown, MessageCircle, Share2, ArrowLeft, Send } from 'lucide-react';
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

function CommentItem({ comment, depth = 0, currentUser, postId, onReload }) {
    const { toast } = useToast();
    const [showReply, setShowReply] = useState(false);
    const [replyText, setReplyText] = useState('');
    const [userVote, setUserVote] = useState(null);
    const [score, setScore] = useState(comment.score || 0);

    const handleVote = (value) => {
        if (!currentUser) { toast({ title: 'Войдите, чтобы голосовать' }); return; }
        const newVote = userVote === value ? null : value;
        setUserVote(newVote);
        setScore(prev => prev + ((newVote || 0) - (userVote || 0)));
        if (newVote !== null) {
            base44.entities.Vote.create({ user_id: currentUser.id, target_id: comment.id, target_type: 'comment', value });
        }
    };

    const handleReply = async () => {
        if (!replyText.trim()) return;
        await base44.entities.Comment.create({
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

    return (
        <div className={`${depth > 0 ? 'ml-4 border-l-2 border-border/30 pl-3' : ''}`}>
            <div className="py-2.5">
                <div className="flex items-center gap-2 mb-1">
                    <img src={comment.author_avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${comment.author_username}`} className="w-6 h-6 rounded-full object-cover" alt="" />
                    <Link to={`/user/${comment.author_id}`} className="text-xs font-bold hover:text-primary">{comment.author_username}</Link>
                    {comment.author_level && <Badge className="text-[9px] h-4 px-1.5 bg-primary/10 text-primary border-0">Ур. {comment.author_level}</Badge>}
                    <span className="text-[10px] text-muted-foreground ml-auto">{timeAgoShort(comment.created_date)}</span>
                </div>

                {comment.is_deleted ? (
                    <p className="text-xs text-muted-foreground italic ml-8">[Удалено]</p>
                ) : (
                    <p className="text-sm text-foreground leading-relaxed ml-8">{comment.content}</p>
                )}

                <div className="flex items-center gap-2 mt-1.5 ml-8">
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
        </div>
    );
}

export default function PostPage() {
    const { id } = useParams();
    const { user } = useAuth();
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

    useEffect(() => { loadPost(); }, [id]);

    const loadPost = async () => {
        setLoading(true);
        const [posts, commentsData] = await Promise.all([
            base44.entities.Post.filter({ id }),
            base44.entities.Comment.filter({ post_id: id }, 'created_date'),
        ]);
        if (posts[0]) {
            setPost(posts[0]);
            setScore(posts[0].score || 0);
            base44.entities.Post.update(posts[0].id, { views: (posts[0].views || 0) + 1 });
        }
        setComments(commentsData);
        setLoading(false);
    };

    const handleVote = (value) => {
        if (!user) { toast({ title: 'Войдите для голосования' }); return; }
        const newVote = userVote === value ? null : value;
        const delta = (newVote || 0) - (userVote || 0);
        setUserVote(newVote);
        setScore(prev => prev + delta);
        // Fire and forget — no reload
        if (newVote !== null) {
            base44.entities.Vote.create({ user_id: user.id, target_id: post.id, target_type: 'post', value });
        }
        base44.entities.Post.update(post.id, { score: score + delta });
    };

    const handleComment = async () => {
        if (!newComment.trim() || !user) return;
        setSubmittingComment(true);
        await base44.entities.Comment.create({
            post_id: id,
            author_id: user.id,
            author_username: user.full_name || user.email,
            author_avatar: user.avatar_url,
            content: newComment.trim(),
            score: 0,
            depth: 0,
        });
        await base44.entities.Post.update(id, { comment_count: (post.comment_count || 0) + 1 });
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
            <CommentItem comment={comment} depth={comment.depth || 0} currentUser={user} postId={id} onReload={loadPost} />
            {comment.children?.length > 0 && <div>{renderComments(comment.children)}</div>}
            <div className="border-b border-border/20 last:border-0" />
        </div>
    ));

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;
    if (!post) return <EmptyState icon={MessageCircle} title="Публикация не найдена" />;

    const topLevelComments = buildCommentTree(sortedComments);

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
                <div className="flex items-center gap-2 px-4 pt-3 pb-1.5">
                    <Link to={`/community/${post.community_id}`} className="flex items-center gap-1.5 group">
                        <img src={post.community_avatar || `https://api.dicebear.com/7.x/shapes/svg?seed=${post.community_name}`} className="w-5 h-5 rounded object-cover" alt="" />
                        <span className="text-xs font-bold text-primary group-hover:underline">{post.community_name}</span>
                    </Link>
                    <span className="text-muted-foreground text-xs">·</span>
                    <Link to={`/user/${post.author_id}`} className="flex items-center gap-1.5 group">
                        <img src={post.author_avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${post.author_username}`} className="w-4 h-4 rounded-full" alt="" />
                        <span className="text-xs text-muted-foreground group-hover:text-foreground">{post.author_username}</span>
                    </Link>
                    <span className="text-muted-foreground text-xs ml-auto">{timeAgoShort(post.created_date)}</span>
                </div>

                <h1 className="px-4 text-xl font-display font-black text-foreground leading-snug mb-2">{post.title}</h1>

                {post.tags?.length > 0 && (
                    <div className="flex flex-wrap gap-1.5 px-4 mb-2">
                        {post.tags.map(tag => <Badge key={tag} className="bg-primary/10 text-primary border-0 text-xs rounded-full">#{tag}</Badge>)}
                    </div>
                )}

                {post.content && <p className="px-4 text-sm text-foreground leading-relaxed mb-3 whitespace-pre-wrap">{post.content}</p>}

                {post.media_urls?.length > 0 && (
                    <div className={`mx-4 mb-3 overflow-hidden rounded ${post.media_urls.length > 1 ? 'grid grid-cols-2 gap-0.5' : ''}`}>
                        {post.media_urls.map((url, i) => <img key={i} src={url} className="w-full object-cover max-h-96" alt="" />)}
                    </div>
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
        </div>
    );
}