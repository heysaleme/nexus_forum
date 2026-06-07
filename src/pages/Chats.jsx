import { useState, useEffect, useRef } from 'react';
import { base44 } from '@/api/base44Client';
import { useAuth } from '@/lib/AuthContext';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import EmptyState from '@/components/ui/EmptyState';
import { MessageCircle, Send, Search, Plus, ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { motion, AnimatePresence } from 'framer-motion';
import { formatDistanceToNow } from 'date-fns';
import { ru } from 'date-fns/locale';
import { Link } from 'react-router-dom';

function ChatBubble({ message, isOwn }) {
    return (
        <motion.div
            initial={{ opacity: 0, y: 8, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            className={`flex items-end gap-2 ${isOwn ? 'flex-row-reverse' : 'flex-row'}`}
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
        </motion.div>
    );
}

export default function Chats() {
    const { user } = useAuth();
    const [rooms, setRooms] = useState([]);
    const [selectedRoom, setSelectedRoom] = useState(null);
    const [messages, setMessages] = useState([]);
    const [newMsg, setNewMsg] = useState('');
    const [loading, setLoading] = useState(true);
    const [sending, setSending] = useState(false);
    const messagesEndRef = useRef(null);

    useEffect(() => {
        if (user) loadRooms();
    }, [user]);

    useEffect(() => {
        if (selectedRoom) {
            markRoomAsRead(selectedRoom.id);
            loadMessages(selectedRoom.id);
            const unsub = base44.entities.Message.subscribe(event => {
                if (event.data?.chat_room_id === selectedRoom.id) {
                    setMessages(prev => {
                        if (event.type === 'create') return [...prev, event.data];
                        return prev;
                    });
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
        const data = await base44.entities.ChatRoom.filter({ participants: user.id }, '-last_message_at');
        setRooms(data);
        setLoading(false);
    };

    const loadMessages = async (roomId) => {
        const data = await base44.entities.Message.filter({ chat_room_id: roomId }, 'created_date', 50);
        setMessages(data);
    };

    const markRoomAsRead = async (roomId) => {
        await base44.entities.ChatRoom.update(roomId, { unread_count: 0 });
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
        await base44.entities.Message.create(msg);
        await base44.entities.ChatRoom.update(selectedRoom.id, { last_message: newMsg.trim(), last_message_at: new Date().toISOString() });
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
                        <Button variant="ghost" size="icon" className="h-8 w-8 rounded-xl">
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
                                        src={room.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${room.name}`}
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
                                        <p className="text-sm font-bold truncate">{room.name || 'Чат'}</p>
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
                                src={selectedRoom.avatar_url || `https://api.dicebear.com/7.x/avataaars/svg?seed=${selectedRoom.name}`}
                                className="w-9 h-9 rounded-2xl object-cover"
                                alt=""
                            />
                            <div>
                                <p className="text-sm font-bold">{selectedRoom.name || 'Чат'}</p>
                                <p className="text-xs text-muted-foreground">{selectedRoom.type === 'group' ? 'Групповой чат' : 'Личный чат'}</p>
                            </div>
                        </div>

                        {/* Messages */}
                        <div className="flex-1 overflow-y-auto p-4 space-y-3">
                            {messages.map(msg => (
                                <ChatBubble key={msg.id} message={msg} isOwn={msg.sender_id === user.id} />
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
        </div>
    );
}
