import { useState, useEffect, useRef } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { MessageCircle, Send, Search, Plus, ArrowLeft, Trash2, Paperclip, X, FileText, Edit2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { motion, AnimatePresence } from 'framer-motion';
import { formatDistanceToNow, format } from 'date-fns';
import { ru } from 'date-fns/locale';
import { Link, useSearchParams } from 'react-router-dom';
import { useToast } from '@/components/ui/use-toast';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogDescription,
} from "@/components/ui/dialog";

function ChatBubble({ message, isOwn, onEdit, onDelete, canEdit, canDelete }) {
    const [isHovered, setIsHovered] = useState(false);

    const formatTime = (dateStr) => {
        if (!dateStr) return '';
        try {
            return format(new Date(dateStr), 'HH:mm');
        } catch {
            return '';
        }
    };

    return (
        <motion.div
            initial={{ opacity: 0, y: 8, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            className={`flex items-end gap-2 ${isOwn ? 'flex-row-reverse' : 'flex-row'}`}
            onMouseEnter={() => setIsHovered(true)}
            onMouseLeave={() => setIsHovered(false)}
        >
            {!isOwn && (
                <img
                    src={message.sender_avatar || `https://api.dicebear.com/7.x/avataaars/svg?seed=${message.sender_id}`}
                    className="w-7 h-7 rounded-full object-cover flex-shrink-0"
                    alt=""
                />
            )}
            <div className={`max-w-[75%] px-3.5 py-2.5 rounded-2xl text-sm flex flex-col gap-0.5 ${isOwn
                ? 'nexus-gradient text-white rounded-br-sm shadow-nexus'
                : 'bg-muted text-foreground rounded-bl-sm'
                }`}>

                {/* Render Attachment if present */}
                {message.attachment_url && message.attachment_type === 'image' && (
                    <div className="mb-1 rounded-lg overflow-hidden border border-white/10 max-w-sm">
                        <img
                            src={`${nexusApi.BASE_URL.replace('/api', '')}${message.attachment_url}`}
                            alt=""
                            className="max-h-60 w-full object-cover cursor-pointer hover:scale-102 transition-transform"
                            onClick={() => window.open(`${nexusApi.BASE_URL.replace('/api', '')}${message.attachment_url}`, '_blank')}
                        />
                    </div>
                )}
                {message.attachment_url && message.attachment_type !== 'image' && (
                    <a
                        href={`${nexusApi.BASE_URL.replace('/api', '')}${message.attachment_url}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className={`flex items-center gap-2 mb-1 p-2 rounded-xl text-xs border ${isOwn
                            ? 'bg-white/10 border-white/20 text-white'
                            : 'bg-background border-border text-foreground hover:bg-muted/50'
                            }`}
                    >
                        <FileText className="w-4 h-4 flex-shrink-0" />
                        <span className="truncate max-w-[150px]">{message.attachment_url.split('_').slice(1).join('_') || 'Файл'}</span>
                    </a>
                )}

                <div className="break-words">{message.content}</div>
                <div className="text-[9px] opacity-70 self-end flex items-center gap-1 mt-0.5 select-none">
                    {formatTime(message.created_date)}
                    {message.is_edited && <span className="text-[9px] opacity-60 ml-0.5">(изм.)</span>}
                    {isOwn && (
                        message.is_read
                            ? <span className="text-sky-300 font-bold" title="Прочитано">✓✓</span>
                            : message.is_delivered
                                ? <span className="text-white/80" title="Доставлено">✓✓</span>
                                : <span className="text-white/60" title="Отправлено">✓</span>
                    )}
                </div>
            </div>
            {isHovered && (
                <div className="flex flex-col gap-0.5 mb-1 flex-shrink-0">
                    {canEdit && (
                        <button
                            onClick={() => onEdit(message)}
                            className="p-1 text-muted-foreground hover:bg-muted hover:text-foreground rounded-full transition-colors"
                            title="Редактировать"
                        >
                            <Edit2 className="w-3.5 h-3.5" />
                        </button>
                    )}
                    {canDelete && (
                        <button
                            onClick={() => onDelete(message.id)}
                            className="p-1 text-destructive hover:bg-destructive/10 rounded-full transition-colors"
                            title="Удалить"
                        >
                            <Trash2 className="w-3.5 h-3.5" />
                        </button>
                    )}
                </div>
            )}
        </motion.div>
    );
}

export default function Chats() {
    const { user } = useAuth();
    const { toast } = useToast();
    const [searchParams] = useSearchParams();
    const userIdParam = searchParams.get('userId');

    const [rooms, setRooms] = useState([]);
    const [selectedRoom, setSelectedRoom] = useState(null);
    const [messages, setMessages] = useState([]);
    const [newMsg, setNewMsg] = useState('');
    const [loading, setLoading] = useState(true);
    const [sending, setSending] = useState(false);
    const messagesEndRef = useRef(null);

    // Dialogue State
    const [userListOpen, setUserListOpen] = useState(false);
    const [usersList, setUsersList] = useState([]);
    const [searchQuery, setSearchQuery] = useState('');
    // Cache of userId -> userInfo for display
    const [userCache, setUserCache] = useState({});

    const wsRef = useRef(null);
    const [isOtherTyping, setIsOtherTyping] = useState(false);
    const typingTimeoutRef = useRef(null);
    const [typingState, setTypingState] = useState(false);
    const selfTypingTimeoutRef = useRef(null);

    // Phase 2A.2 States & Refs
    const [editingMessage, setEditingMessage] = useState(null);
    const [attachment, setAttachment] = useState(null);
    const [uploading, setUploading] = useState(false);
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
    const [messageToDelete, setMessageToDelete] = useState(null);
    const fileInputRef = useRef(null);

    useEffect(() => {
        if (user) {
            loadRooms();
        }
    }, [user]);

    useEffect(() => {
        if (user && userIdParam) {
            autoStartChat(parseInt(userIdParam));
        }
    }, [user, userIdParam, rooms.length]);

    useEffect(() => {
        if (!selectedRoom?.id) return;
        console.log("WS EFFECT START", selectedRoom?.id);
        return () => {
            console.log("WS EFFECT CLEANUP", selectedRoom?.id);
        };
    }, [selectedRoom?.id]);

    useEffect(() => {
        if (!selectedRoom) return;

        // Reset states
        setIsOtherTyping(false);
        setTypingState(false);
        if (selfTypingTimeoutRef.current) clearTimeout(selfTypingTimeoutRef.current);

        // Load historical messages via HTTP
        markRoomAsRead(selectedRoom.id);
        loadMessages(selectedRoom.id);
        setEditingMessage(null);
        setAttachment(null);

        // Connect to WebSocket dynamically deriving host from BASE_URL
        const token = localStorage.getItem('nexus_forum_session_token');
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const apiBase = nexusApi.BASE_URL;
        let wsHost;
        if (apiBase.startsWith('http')) {
            try {
                const url = new URL(apiBase);
                wsHost = url.host;
            } catch {
                wsHost = window.location.host;
            }
        } else {
            wsHost = window.location.host;
        }
        const wsUrl = `${wsProtocol}//${wsHost}/api/ws/chat/${selectedRoom.id}`;
        const ws = new WebSocket(wsUrl, ["Bearer", token]);

        ws.onopen = () => {
            console.log("WebSocket connected to chat room:", selectedRoom.id);
            // Send initial read receipt for the last message in room
            sendInitialReadReceipt(ws);
        };

        ws.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);

                if (msg.type === 'message') {

                    console.log("WS MESSAGE RAW", JSON.stringify(msg, null, 2));
                }

                if (msg.room_id !== selectedRoom.id && msg.type !== 'online_status') return;

                if (msg.type === 'message') {
                    setMessages(prev => {
                        if (prev.some(m => m.id === msg.id)) return prev;
                        const newMsgObj = {
                            id: msg.id,
                            chat_room_id: msg.room_id,
                            sender_id: msg.sender_id,
                            sender_username: msg.sender_name,
                            content: msg.content,
                            is_read: msg.is_read,
                            is_delivered: msg.is_delivered,
                            attachment_url: msg.attachment_url,
                            attachment_type: msg.attachment_type,
                            created_date: msg.timestamp || new Date().toISOString()
                        };
                        return [...prev, newMsgObj];
                    });

                    // Send read status back to server if message is from the other user
                    if (msg.sender_id !== user.id) {
                        ws.send(JSON.stringify({
                            type: 'read',
                            content: String(msg.id)
                        }));
                    }
                } else if (msg.type === 'message_edited') {
                    setMessages(prev => prev.map(m => {
                        if (m.id === msg.message_id) {
                            return { ...m, content: msg.content, is_edited: msg.is_edited };
                        }
                        return m;
                    }));
                } else if (msg.type === 'message_deleted') {
                    setMessages(prev => prev.map(m => {
                        if (m.id === msg.message_id) {
                            return { ...m, content: msg.content, attachment_url: '', attachment_type: '' };
                        }
                        return m;
                    }));
                } else if (msg.type === 'typing') {
                    if (msg.sender_id !== user.id) {
                        setIsOtherTyping(msg.content === 'true');
                        if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
                        if (msg.content === 'true') {
                            typingTimeoutRef.current = setTimeout(() => {
                                setIsOtherTyping(false);
                            }, 3000);
                        }
                    }
                } else if (msg.type === 'read') {
                    const readUpToId = parseInt(msg.content);
                    setMessages(prev => prev.map(m => {
                        if (m.id <= readUpToId && m.sender_id === user.id) {
                            return { ...m, is_read: true };
                        }
                        return m;
                    }));
                } else if (msg.type === 'online_status') {
                    const status = msg.content;
                    const otherId = msg.sender_id;
                    setUserCache(prev => {
                        if (prev[otherId]) {
                            return {
                                ...prev,
                                [otherId]: {
                                    ...prev[otherId],
                                    is_online: status === 'online',
                                    last_seen_at: status === 'offline' ? new Date().toISOString() : prev[otherId].last_seen_at
                                }
                            };
                        }
                        return prev;
                    });
                }
            } catch (err) {
                console.error("Failed to process WebSocket event:", err);
            }
        };

        ws.onclose = () => {
            console.log("WebSocket connection closed for chat room:", selectedRoom.id);
        };

        wsRef.current = ws;

        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
            if (typingTimeoutRef.current) {
                clearTimeout(typingTimeoutRef.current);
            }
        };
    }, [selectedRoom?.id]);

    const sendInitialReadReceipt = (socket) => {
        if (socket && socket.readyState === WebSocket.OPEN && messages.length > 0) {
            const otherMessages = messages.filter(m => m.sender_id !== user.id);
            if (otherMessages.length > 0) {
                const lastOtherMsg = otherMessages[otherMessages.length - 1];
                if (!lastOtherMsg.is_read) {
                    socket.send(JSON.stringify({
                        type: 'read',
                        content: String(lastOtherMsg.id)
                    }));
                }
            }
        }
    };

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const loadRooms = async () => {
        setLoading(true);
        const data = await nexusApi.entities.ChatRoom.filter({ participants: user.id }, '-last_message_at') || [];
        setRooms(data || []);
        // Pre-cache all participant user info
        const allIds = new Set();
        data.forEach(room => {
            try {
                const pids = JSON.parse(room.participants || '[]');
                pids.forEach(id => { if (id !== user.id) allIds.add(id); });
            } catch { }
        });
        const cacheUpdates = {};
        await Promise.all([...allIds].map(async (uid) => {
            const users = await nexusApi.entities.User.filter({ id: uid }).catch(() => []);
            if (users[0]) cacheUpdates[uid] = users[0];
        }));
        setUserCache(cacheUpdates);
        setLoading(false);
    };

    const getOtherUserId = (room) => {
        try {
            const pids = JSON.parse(room.participants || '[]');
            return pids.find(id => id !== user.id);
        } catch { }
        return null;
    };

    const isRoomOnline = (room) => {
        const otherId = getOtherUserId(room);
        return otherId && userCache[otherId]?.is_online;
    };

    const getOnlineStatusText = (room) => {
        const otherId = getOtherUserId(room);
        if (otherId && userCache[otherId]) {
            const u = userCache[otherId];
            if (u.is_online) return 'В сети';
            if (u.last_seen_at) {
                try {
                    return `Был(а) в сети ${formatDistanceToNow(new Date(u.last_seen_at), { locale: ru, addSuffix: true })}`;
                } catch {
                    return 'Не в сети';
                }
            }
        }
        return room.type === 'group' ? 'Групповой чат' : 'Не в сети';
    };

    // Returns the OTHER user's name for a direct chat room
    const getRoomDisplayName = (room) => {
        try {
            const pids = JSON.parse(room.participants || '[]');
            const otherId = pids.find(id => id !== user.id);
            if (otherId && userCache[otherId]) {
                return userCache[otherId].username || userCache[otherId].full_name || 'Пользователь';
            }
        } catch { }
        return room.name || 'Чат';
    };

    const getRoomAvatar = (room) => {
        try {
            const pids = JSON.parse(room.participants || '[]');
            const otherId = pids.find(id => id !== user.id);
            if (otherId && userCache[otherId]) {
                return userCache[otherId].avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${otherId}`;
            }
        } catch { }
        return `https://api.dicebear.com/7.x/avataaars/svg?seed=${room.name}`;
    };

    const autoStartChat = async (targetUserId) => {
        try {
            let existingRoom = null;
            for (const room of rooms) {
                try {
                    const pids = JSON.parse(room.participants);
                    if (pids.length === 2 && pids.includes(targetUserId) && pids.includes(user.id)) {
                        existingRoom = room;
                        break;
                    }
                } catch (e) { }
            }

            if (existingRoom) {
                setSelectedRoom(existingRoom);
                return;
            }

            // If not found in current loaded rooms, double check/create it
            const targetUsers = await nexusApi.entities.User.filter({ id: targetUserId });
            if (targetUsers[0]) {
                const targetUser = targetUsers[0];
                const name = targetUser.full_name || targetUser.username;
                const newRoom = await nexusApi.entities.ChatRoom.create({
                    participants: [user.id, targetUser.id],
                    name: name,
                });
                const updatedRooms = await nexusApi.entities.ChatRoom.filter({ participants: user.id }, '-last_message_at');
                setRooms(updatedRooms);
                const found = updatedRooms.find(r => r.participants && JSON.parse(r.participants).includes(targetUserId));
                setSelectedRoom(found || newRoom);
            }
        } catch (err) {
            console.error("Auto start chat failed", err);
        }
    };

    const openNewChatDialog = async () => {
        setUserListOpen(true);
        try {
            // Show only users that the current user is following
            const following = await nexusApi.entities.UserFollow.filter({ follower_id: user.id });
            if (following && following.length > 0) {
                setUsersList(following.filter(u => u.id !== user.id));
            } else {
                // Fallback: show all users if no follows (shouldn't happen normally)
                const list = await nexusApi.entities.User.list('-created_date', 50);
                setUsersList(list.filter(u => u.id !== user.id));
            }
        } catch (err) {
            console.error(err);
            const list = await nexusApi.entities.User.list('-created_date', 50);
            setUsersList(list.filter(u => u.id !== user.id));
        }
    };

    const handleStartChat = async (targetUser) => {
        try {
            let existingRoom = null;
            for (const room of rooms) {
                try {
                    const pids = JSON.parse(room.participants);
                    if (pids.length === 2 && pids.includes(targetUser.id) && pids.includes(user.id)) {
                        existingRoom = room;
                        break;
                    }
                } catch (e) { }
            }

            if (existingRoom) {
                setSelectedRoom(existingRoom);
                setUserListOpen(false);
                return;
            }

            // Name the room as the OTHER person's name (not self)
            const targetName = targetUser.username || targetUser.full_name || 'Пользователь';
            const newRoom = await nexusApi.entities.ChatRoom.create({
                participants: [user.id, targetUser.id],
                name: targetName,
            });
            // Cache the target user
            setUserCache(prev => ({ ...prev, [targetUser.id]: targetUser }));
            setUserListOpen(false);
            const updatedRooms = await nexusApi.entities.ChatRoom.filter({ participants: user.id }, '-last_message_at');
            setRooms(updatedRooms);
            const found = updatedRooms.find(r => r.participants && JSON.parse(r.participants).includes(targetUser.id));
            setSelectedRoom(found || newRoom);
            toast({ title: `✅ Диалог с ${targetName} создан!` });
        } catch (err) {
            toast({ title: 'Не удалось создать чат', variant: 'destructive' });
        }
    };

    const handleDeleteMessage = (msgId) => {
        const msg = messages.find(m => m.id === msgId);
        if (!msg) return;
        setMessageToDelete(msg);
        setDeleteConfirmOpen(true);
    };

    const handleConfirmDelete = async (deleteType) => {
        if (!messageToDelete) return;
        try {
            await nexusApi.entities.Message.delete(messageToDelete.id, deleteType);
            if (deleteType === 'me') {
                setMessages(prev => prev.filter(m => m.id !== messageToDelete.id));
            } else {
                setMessages(prev => prev.map(m => {
                    if (m.id === messageToDelete.id) {
                        return { ...m, content: 'Сообщение удалено', attachment_url: '', attachment_type: '' };
                    }
                    return m;
                }));
            }
            setDeleteConfirmOpen(false);
            setMessageToDelete(null);
            toast({ title: '🗑️ Сообщение удалено' });
        } catch (err) {
            toast({ title: 'Не удалось удалить сообщение', variant: 'destructive' });
        }
    };

    const handleStartEdit = (msg) => {
        setEditingMessage(msg);
        setNewMsg(msg.content);
        setAttachment(null);
    };

    const handleAttachmentSelect = async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        if (file.size > 10 * 1024 * 1024) {
            toast({ title: 'Размер файла превышает 10MB', variant: 'destructive' });
            return;
        }

        const formData = new FormData();
        formData.append('file', file);

        setUploading(true);
        try {
            const token = localStorage.getItem('nexus_forum_session_token');
            const response = await fetch(`${nexusApi.BASE_URL}/upload`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                },
                body: formData
            });

            if (!response.ok) {
                const errData = await response.json().catch(() => ({}));
                throw new Error(errData.error || 'Failed to upload');
            }

            const data = await response.json();
            const type = data.mime_type.startsWith('image/') ? 'image' : 'file';
            setAttachment({
                url: data.url,
                name: data.filename,
                type: type
            });
            toast({ title: '📎 Файл загружен' });
        } catch (err) {
            toast({ title: 'Не удалось загрузить файл: ' + err.message, variant: 'destructive' });
        } finally {
            setUploading(false);
        }
    };

    const handleDeleteRoom = async (roomId) => {
        if (!window.confirm('Вы уверены, что хотите удалить весь диалог для обоих участников? Это действие сотрет всю переписку.')) return;
        try {
            await nexusApi.entities.ChatRoom.delete(roomId);
            toast({ title: '🗑️ Диалог успешно удален' });
            setSelectedRoom(null);
            setRooms(prev => prev.filter(r => r.id !== roomId));
        } catch (err) {
            toast({ title: 'Не удалось удалить диалог', variant: 'destructive' });
        }
    };

    const loadMessages = async (roomId) => {
        const data = await nexusApi.entities.Message.filter(
            { chat_room_id: roomId },
            'created_date',
            50
        );

        setMessages(data || []);
    };

    const markRoomAsRead = async (roomId) => {
        console.log("MARK ROOM READ", roomId);

        await nexusApi.entities.ChatRoom.update(roomId, { unread_count: 0 });
        setRooms(prev => prev.map(room => room.id === roomId ? { ...room, unread_count: 0 } : room));

    };

    const handleInputChange = (e) => {
        setNewMsg(e.target.value);

        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            if (!typingState) {
                setTypingState(true);
                wsRef.current.send(JSON.stringify({
                    type: 'typing',
                    content: 'true'
                }));
            }

            if (selfTypingTimeoutRef.current) clearTimeout(selfTypingTimeoutRef.current);
            selfTypingTimeoutRef.current = setTimeout(() => {
                setTypingState(false);
                if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
                    wsRef.current.send(JSON.stringify({
                        type: 'typing',
                        content: 'false'
                    }));
                }
            }, 2000);
        }
    };

    const handleSend = async () => {
        if ((!newMsg.trim() && !attachment) || !selectedRoom) return;

        // Reset typing indicators
        if (selfTypingTimeoutRef.current) clearTimeout(selfTypingTimeoutRef.current);
        setTypingState(false);

        // 1. Edit Mode
        if (editingMessage) {
            setSending(true);
            try {
                await nexusApi.entities.Message.update(editingMessage.id, newMsg.trim());
                setNewMsg('');
                setEditingMessage(null);
            } catch (err) {
                toast({ title: 'Не удалось отредактировать сообщение', variant: 'destructive' });
            } finally {
                setSending(false);
            }
            return;
        }

        // 2. Attachment Send Mode (always via HTTP POST API)
        if (attachment) {
            setSending(true);
            try {
                await nexusApi.entities.Message.create({
                    chat_room_id: selectedRoom.id,
                    content: newMsg.trim(),
                    attachment_url: attachment.url,
                    attachment_type: attachment.type
                });
                setNewMsg('');
                setAttachment(null);
            } catch (err) {
                toast({ title: 'Не удалось отправить сообщение', variant: 'destructive' });
            } finally {
                setSending(false);
            }
            return;
        }

        // 3. WS Text Send Mode
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({
                type: 'message',
                content: newMsg.trim()
            }));

            wsRef.current.send(JSON.stringify({
                type: 'typing',
                content: 'false'
            }));

            // Optimistic update of chat room sidebar details
            setRooms(prev => prev.map(room => {
                if (room.id === selectedRoom.id) {
                    return {
                        ...room,
                        last_message: newMsg.trim(),
                        last_message_at: new Date().toISOString()
                    };
                }
                return room;
            }));

            setNewMsg('');
        } else {
            // 4. HTTP Text Send fallback
            setSending(true);
            try {
                const msg = {
                    sender_id: user.id,
                    sender_username: user.full_name || user.email,
                    sender_avatar: user.avatar_url,
                    chat_room_id: selectedRoom.id,
                    content: newMsg.trim(),
                    message_type: 'text',
                    is_read: false,
                };
                await nexusApi.entities.Message.create(msg);
                await nexusApi.entities.ChatRoom.update(selectedRoom.id, { last_message: newMsg.trim(), last_message_at: new Date().toISOString() });
                setNewMsg('');
                loadMessages(selectedRoom.id);
            } catch (err) {
                toast({ title: 'Не удалось отправить сообщение', variant: 'destructive' });
            } finally {
                setSending(false);
            }
        }
    };

    if (!user) {
        return (
            <EmptyState icon={MessageCircle} title="Войдите для доступа к чатам"
                action={<Link to="/login"><Button className="nexus-gradient border-0 text-white rounded-xl shadow-nexus">Войти</Button></Link>}
            />
        );
    }

    useEffect(() => {
        console.table(
            messages.map(m => ({
                id: m.id,
                sender: m.sender_id,
                content: m.content
            }))
        );
    }, [messages]);

    return (
        <div className="flex h-[calc(100vh-4rem)] md:h-[calc(100vh-3.5rem)]">
            {/* Sidebar */}
            <div className={`w-full md:w-80 border-r border-border/50 flex flex-col ${selectedRoom ? 'hidden md:flex' : 'flex'}`}>
                <div className="p-3 border-b border-border/50">
                    <div className="flex items-center justify-between mb-2">
                        <h2 className="font-display font-black text-base">Чаты</h2>
                        <Button variant="ghost" size="icon" onClick={openNewChatDialog} className="h-8 w-8 rounded-xl">
                            <Plus className="w-4 h-4" />
                        </Button>
                    </div>
                    <div className="relative">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
                        <Input placeholder="Поиск чатов..." className="pl-8 h-8 text-xs rounded-xl bg-muted/50 border-0" />
                    </div>
                </div>

                {loading ? <LoadingSpinner className="py-8" /> : (rooms || []).length === 0 ? (
                    <EmptyState icon={MessageCircle} title="Нет диалогов" description="Начни общение с другими пользователями" />
                ) : (
                    <div className="flex-1 overflow-y-auto">
                        {(rooms || []).map(room => (
                            <button
                                key={room.id}
                                onClick={() => setSelectedRoom(room)}
                                className={`w-full flex items-center gap-3 p-3 hover:bg-muted/50 transition-colors text-left ${selectedRoom?.id === room.id ? 'bg-primary/10' : ''}`}
                            >
                                <div className="relative flex-shrink-0">
                                    <img
                                        src={getRoomAvatar(room)}
                                        className="w-10 h-10 rounded-2xl object-cover"
                                        alt=""
                                    />
                                    {isRoomOnline(room) && (
                                        <div className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 bg-green-500 border-2 border-background rounded-full" title="В сети" />
                                    )}
                                    {room.unread_count > 0 && (
                                        <div className="absolute -top-1 -right-1 w-4 h-4 nexus-gradient rounded-full flex items-center justify-center">
                                            <span className="text-white text-[9px] font-black">{room.unread_count}</span>
                                        </div>
                                    )}
                                </div>
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center justify-between">
                                        <p className="text-sm font-bold truncate">{getRoomDisplayName(room)}</p>
                                        {room.last_message_at && (
                                            <p className="text-[10px] text-muted-foreground flex-shrink-0">
                                                {formatDistanceToNow(new Date(room.last_message_at), { locale: ru })}
                                            </p>
                                        )}
                                    </div>
                                    <p className="text-xs text-muted-foreground truncate">{room.last_message || 'Нет сообщений'}</p>
                                </div>
                            </button>
                        ))}
                    </div>
                )}
            </div>

            {/* Chat area */}
            <div className={`flex-1 flex flex-col ${!selectedRoom ? 'hidden md:flex' : 'flex'}`}>
                {!selectedRoom ? (
                    <div className="flex-1 flex items-center justify-center">
                        <EmptyState icon={MessageCircle} title="Выберите чат" description="Нажмите на диалог слева, чтобы начать общение" />
                    </div>
                ) : (
                    <>
                        {/* Chat header */}
                        <div className="flex items-center gap-3 p-3 border-b border-border/50 bg-card">
                            <Button variant="ghost" size="icon" className="h-8 w-8 rounded-xl md:hidden" onClick={() => setSelectedRoom(null)}>
                                <ArrowLeft className="w-4 h-4" />
                            </Button>
                            <img
                                src={getRoomAvatar(selectedRoom)}
                                className="w-9 h-9 rounded-2xl object-cover"
                                alt=""
                            />
                            <div className="flex-1 min-w-0">
                                <p className="text-sm font-bold">{getRoomDisplayName(selectedRoom)}</p>
                                <p className="text-xs text-muted-foreground transition-all duration-300">
                                    {isOtherTyping
                                        ? <span className="text-primary font-semibold animate-pulse">печатает...</span>
                                        : getOnlineStatusText(selectedRoom)
                                    }
                                </p>
                            </div>
                            <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 rounded-xl text-destructive hover:bg-destructive/10"
                                onClick={() => handleDeleteRoom(selectedRoom.id)}
                            >
                                <Trash2 className="w-4 h-4" />
                            </Button>
                        </div>

                        {/* Messages */}
                        <div className="flex-1 overflow-y-auto p-4 space-y-3">
                            {(messages || []).map(msg => {
                                const canEdit = msg.sender_id === user.id && msg.content !== 'Сообщение удалено';
                                const canDelete = msg.sender_id === user.id || (user && (user.role === 'admin' || user.role === 'moderator'));
                                return (
                                    <ChatBubble
                                        key={msg.id ?? `${msg.sender_id}-${msg.created_date}`}
                                        message={msg}
                                        isOwn={msg.sender_id === user.id}
                                        onEdit={handleStartEdit}
                                        onDelete={handleDeleteMessage}
                                        canEdit={canEdit}
                                        canDelete={canDelete}
                                    />
                                );
                            })}
                            <div ref={messagesEndRef} />
                        </div>

                        {/* Input & Preview Area */}
                        <div className="p-3 border-t border-border/50 bg-card">

                            {/* Editing Message Header */}
                            {editingMessage && (
                                <div className="flex items-center justify-between p-2 bg-primary/10 rounded-xl mb-2 mx-1 border border-primary/20 text-xs">
                                    <div className="flex items-center gap-2 truncate">
                                        <Edit2 className="w-3.5 h-3.5 text-primary flex-shrink-0" />
                                        <span className="font-semibold text-primary">Редактирование:</span>
                                        <span className="truncate text-muted-foreground">{editingMessage.content}</span>
                                    </div>
                                    <button
                                        onClick={() => { setEditingMessage(null); setNewMsg(''); }}
                                        className="p-1 hover:bg-muted rounded-full"
                                    >
                                        <X className="w-3.5 h-3.5 text-muted-foreground" />
                                    </button>
                                </div>
                            )}

                            {/* Upload Preview Area */}
                            {attachment && (
                                <div className="flex items-center gap-3 p-2 bg-muted/60 rounded-xl mb-2 mx-1 border border-border/40 relative">
                                    <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                                        {attachment.type === 'image' ? (
                                            <img src={`${nexusApi.BASE_URL.replace('/api', '')}${attachment.url}`} className="w-10 h-10 rounded-lg object-cover" alt="" />
                                        ) : (
                                            <FileText className="w-5 h-5 text-primary" />
                                        )}
                                    </div>
                                    <div className="flex-1 min-w-0">
                                        <p className="text-xs font-bold truncate">{attachment.name}</p>
                                        <p className="text-[10px] text-muted-foreground uppercase">{attachment.type}</p>
                                    </div>
                                    <button onClick={() => setAttachment(null)} className="p-1 hover:bg-muted rounded-full">
                                        <X className="w-4 h-4 text-muted-foreground" />
                                    </button>
                                </div>
                            )}

                            <div className="flex gap-2">
                                <input
                                    type="file"
                                    ref={fileInputRef}
                                    onChange={handleAttachmentSelect}
                                    className="hidden"
                                />
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={() => fileInputRef.current?.click()}
                                    disabled={uploading || !!editingMessage}
                                    className="h-10 w-10 rounded-xl text-muted-foreground hover:bg-muted flex-shrink-0"
                                >
                                    <Paperclip className="w-5 h-5" />
                                </Button>

                                <Input
                                    value={newMsg}
                                    onChange={handleInputChange}
                                    onKeyDown={e => e.key === 'Enter' && !e.shiftKey && (e.preventDefault(), handleSend())}
                                    placeholder={uploading ? "Загрузка файла..." : "Написать сообщение..."}
                                    disabled={uploading}
                                    className="rounded-xl bg-muted/50 border-0 text-sm h-10 flex-1"
                                />

                                <Button
                                    onClick={handleSend}
                                    disabled={(!newMsg.trim() && !attachment) || sending || uploading}
                                    size="icon"
                                    className="h-10 w-10 nexus-gradient border-0 text-white rounded-xl shadow-nexus flex-shrink-0"
                                >
                                    <Send className="w-4 h-4" />
                                </Button>
                            </div>
                        </div>
                    </>
                )}
            </div>

            {/* New Chat Dialog */}
            <Dialog open={userListOpen} onOpenChange={setUserListOpen}>
                <DialogContent className="sm:max-w-[425px] rounded-2xl p-4 bg-card border border-border">
                    <DialogHeader>
                        <DialogTitle className="font-display font-black text-lg">Начать диалог</DialogTitle>
                    </DialogHeader>

                    <div className="relative my-3">
                        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                        <Input
                            value={searchQuery}
                            onChange={e => setSearchQuery(e.target.value)}
                            placeholder="Поиск пользователей..."
                            className="pl-9 bg-muted/50 border-0 rounded-xl h-10 text-sm w-full"
                        />
                    </div>

                    <div className="max-h-[300px] overflow-y-auto space-y-2 mt-2">
                        {(usersList || []).filter(u => {
                            const query = searchQuery.toLowerCase();
                            return (u.username || '').toLowerCase().includes(query) || (u.full_name || '').toLowerCase().includes(query);
                        }).map(u => (
                            <button
                                key={u.id}
                                onClick={() => handleStartChat(u)}
                                className="w-full flex items-center gap-3 p-2 hover:bg-muted/50 rounded-xl transition-colors text-left"
                            >
                                <img
                                    src={u.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${u.email}`}
                                    className="w-10 h-10 rounded-full object-cover flex-shrink-0"
                                    alt=""
                                />
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-bold truncate">{u.full_name || u.username}</p>
                                    <p className="text-xs text-muted-foreground truncate">@{u.username}</p>
                                </div>
                            </button>
                        ))}
                    </div>
                </DialogContent>
            </Dialog>

            {/* Delete Message Confirmation Dialog */}
            <Dialog open={deleteConfirmOpen} onOpenChange={setDeleteConfirmOpen}>
                <DialogContent className="sm:max-w-[400px] rounded-2xl p-5 bg-card border border-border">
                    <DialogHeader>
                        <DialogTitle className="font-display font-black text-lg">Удалить сообщение</DialogTitle>
                        <DialogDescription className="text-sm">
                            Выберите тип удаления. Вы можете стереть сообщение у себя или для всех участников.
                        </DialogDescription>
                    </DialogHeader>
                    <DialogFooter className="flex flex-col sm:flex-row gap-2 mt-4">
                        <Button variant="ghost" className="rounded-xl order-3 sm:order-1" onClick={() => { setDeleteConfirmOpen(false); setMessageToDelete(null); }}>
                            Отмена
                        </Button>
                        <Button variant="outline" className="rounded-xl order-2" onClick={() => handleConfirmDelete('me')}>
                            Удалить для себя
                        </Button>
                        <Button className="nexus-gradient text-white border-0 rounded-xl order-1 sm:order-3 shadow-nexus" onClick={() => handleConfirmDelete('everyone')}>
                            Удалить для всех
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>
        </div>
    );
}
