import { useState, useEffect, useRef } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { MessageCircle, Send, Search, Plus, ArrowLeft, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { motion, AnimatePresence } from 'framer-motion';
import { formatDistanceToNow } from 'date-fns';
import { ru } from 'date-fns/locale';
import { Link, useSearchParams } from 'react-router-dom';
import { useToast } from '@/components/ui/use-toast';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";

function ChatBubble({ message, isOwn, onDelete }) {
    const [isHovered, setIsHovered] = useState(false);
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
            <div className={`max-w-[75%] px-3.5 py-2.5 rounded-2xl text-sm ${isOwn
                    ? 'nexus-gradient text-white rounded-br-sm shadow-nexus'
                    : 'bg-muted text-foreground rounded-bl-sm'
                }`}>
                {message.content}
            </div>
            {isOwn && isHovered && (
                <button
                    onClick={() => onDelete(message.id)}
                    className="p-1 text-destructive hover:bg-destructive/10 rounded-full transition-colors flex-shrink-0 mb-1"
                    title="Удалить сообщение"
                >
                    <Trash2 className="w-3.5 h-3.5" />
                </button>
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
        if (selectedRoom) {
            markRoomAsRead(selectedRoom.id);
            loadMessages(selectedRoom.id);
            const unsub = nexusApi.entities.Message.subscribe(event => {
                if (event.data?.chat_room_id === selectedRoom.id) {
                    if (event.type === 'create') {
                        setMessages(prev => {
                            if (prev.some(m => m.id === event.data.id)) return prev;
                            return [...prev, event.data];
                        });
                    } else if (event.type === 'delete') {
                        setMessages(prev => prev.filter(m => m.id !== event.data.id));
                    }
                }
            });
            return unsub;
        }
    }, [selectedRoom]);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const loadRooms = async () => {
        setLoading(true);
        const data = await nexusApi.entities.ChatRoom.filter({ participants: user.id }, '-last_message_at');
        setRooms(data);
        // Pre-cache all participant user info
        const allIds = new Set();
        data.forEach(room => {
            try {
                const pids = JSON.parse(room.participants || '[]');
                pids.forEach(id => { if (id !== user.id) allIds.add(id); });
            } catch {}
        });
        const cacheUpdates = {};
        await Promise.all([...allIds].map(async (uid) => {
            const users = await nexusApi.entities.User.filter({ id: uid }).catch(() => []);
            if (users[0]) cacheUpdates[uid] = users[0];
        }));
        setUserCache(cacheUpdates);
        setLoading(false);
    };

    // Returns the OTHER user's name for a direct chat room
    const getRoomDisplayName = (room) => {
        try {
            const pids = JSON.parse(room.participants || '[]');
            const otherId = pids.find(id => id !== user.id);
            if (otherId && userCache[otherId]) {
                return userCache[otherId].username || userCache[otherId].full_name || 'Пользователь';
            }
        } catch {}
        return room.name || 'Чат';
    };

    const getRoomAvatar = (room) => {
        try {
            const pids = JSON.parse(room.participants || '[]');
            const otherId = pids.find(id => id !== user.id);
            if (otherId && userCache[otherId]) {
                return userCache[otherId].avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${otherId}`;
            }
        } catch {}
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
                } catch (e) {}
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
                } catch (e) {}
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

    const handleDeleteMessage = async (msgId) => {
        if (!window.confirm('Вы уверены, что хотите удалить это сообщение?')) return;
        try {
            await nexusApi.entities.Message.delete(msgId);
            setMessages(prev => prev.filter(m => m.id !== msgId));
            toast({ title: '🗑️ Сообщение удалено' });
        } catch (err) {
            toast({ title: 'Не удалось удалить сообщение', variant: 'destructive' });
        }
    };

    const loadMessages = async (roomId) => {
        const data = await nexusApi.entities.Message.filter({ chat_room_id: roomId }, 'created_date', 50);
        setMessages(data);
    };

    const markRoomAsRead = async (roomId) => {
        await nexusApi.entities.ChatRoom.update(roomId, { unread_count: 0 });
        setRooms(prev => prev.map(room => room.id === roomId ? { ...room, unread_count: 0 } : room));
        setSelectedRoom(prev => prev?.id === roomId ? { ...prev, unread_count: 0 } : prev);
    };

    const handleSend = async () => {
        if (!newMsg.trim() || !selectedRoom || sending) return;
        setSending(true);
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
        setSending(false);
        loadMessages(selectedRoom.id);
    };

    if (!user) {
        return (
            <EmptyState icon={MessageCircle} title="Войдите для доступа к чатам"
                action={<Link to="/login"><Button className="nexus-gradient border-0 text-white rounded-xl shadow-nexus">Войти</Button></Link>}
            />
        );
    }

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

                {loading ? <LoadingSpinner className="py-8" /> : rooms.length === 0 ? (
                    <EmptyState icon={MessageCircle} title="Нет диалогов" description="Начни общение с другими пользователями" />
                ) : (
                    <div className="flex-1 overflow-y-auto">
                        {rooms.map(room => (
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
                            <div>
                                <p className="text-sm font-bold">{getRoomDisplayName(selectedRoom)}</p>
                                <p className="text-xs text-muted-foreground">{selectedRoom.type === 'group' ? 'Групповой чат' : 'Личный чат'}</p>
                            </div>
                        </div>

                        {/* Messages */}
                        <div className="flex-1 overflow-y-auto p-4 space-y-3">
                            {messages.map(msg => (
                                <ChatBubble key={msg.id} message={msg} isOwn={msg.sender_id === user.id} onDelete={handleDeleteMessage} />
                            ))}
                            <div ref={messagesEndRef} />
                        </div>

                        {/* Input */}
                        <div className="p-3 border-t border-border/50 bg-card">
                            <div className="flex gap-2">
                                <Input
                                    value={newMsg}
                                    onChange={e => setNewMsg(e.target.value)}
                                    onKeyDown={e => e.key === 'Enter' && !e.shiftKey && (e.preventDefault(), handleSend())}
                                    placeholder="Написать сообщение..."
                                    className="rounded-xl bg-muted/50 border-0 text-sm h-10"
                                />
                                <Button
                                    onClick={handleSend}
                                    disabled={!newMsg.trim() || sending}
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
                        {usersList.filter(u => {
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
        </div>
    );
}
