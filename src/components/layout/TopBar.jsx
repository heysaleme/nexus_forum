import { Link, useNavigate } from 'react-router-dom';
import { Search, Bell, Zap, Sun, Moon, Settings } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useState } from 'react';
import { motion } from 'framer-motion';
import useUnreadCounts from '@/hooks/useUnreadCounts';

export default function TopBar({ user }) {
    const navigate = useNavigate();
    const [query, setQuery] = useState('');
    const [dark, setDark] = useState(false);
    const { notifications } = useUnreadCounts(user);

    const handleSearch = (e) => {
        e.preventDefault();
        if (query.trim()) navigate(`/search?q=${encodeURIComponent(query)}`);
    };

    const toggleDark = () => {
        setDark(!dark);
        document.documentElement.classList.toggle('dark');
    };

    return (
        <header className="sticky top-0 z-40 glass border-b border-border/50 px-4 py-3">
            <div className="max-w-7xl mx-auto flex items-center gap-3.5">
                {/* Logo (mobile only) */}
                <Link to="/" className="md:hidden flex items-center gap-2 flex-shrink-0">
                    <div className="w-8 h-8 nexus-gradient rounded-xl flex items-center justify-center">
                        <Zap className="w-4 h-4 text-white" />
                    </div>
                    <span className="font-display font-black text-lg">Nexus</span>
                </Link>

                {/* Search */}
                <form onSubmit={handleSearch} className="flex-1 max-w-2xl mx-auto md:mx-8">
                    <div className="relative">
                        <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                        <Input
                            value={query}
                            onChange={(e) => setQuery(e.target.value)}
                            placeholder="Поиск постов, сообществ, пользователей..."
                            className="pl-11 pr-4 bg-muted/55 border-0 rounded-[1rem] focus-visible:ring-1 focus-visible:ring-primary text-sm h-10"
                        />
                    </div>
                </form>

                <div className="flex items-center gap-2.5 ml-auto mr-2 md:mr-8">
                    <Button variant="ghost" size="icon" className="h-10 w-10 rounded-[1rem]" onClick={toggleDark}>
                        {dark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
                    </Button>

                    {user ? (
                        <>
                            <Link to="/notifications">
                                <Button variant="ghost" size="icon" className="h-10 w-10 rounded-[1rem] relative">
                                    <Bell className="w-4 h-4" />
                                    {notifications > 0 && (
                                        <span className="absolute top-1.5 right-1.5 min-w-4 h-4 px-1 rounded-full bg-primary text-[10px] font-bold text-white flex items-center justify-center">
                                            {notifications > 9 ? '9+' : notifications}
                                        </span>
                                    )}
                                </Button>
                            </Link>
                        </>
                    ) : (
                        <div className="flex gap-2">
                            <Link to="/login">
                                <Button variant="ghost" size="sm" className="rounded-xl text-xs">Войти</Button>
                            </Link>
                            <Link to="/register">
                                <Button size="sm" className="rounded-xl nexus-gradient border-0 text-white text-xs shadow-nexus">
                                    Регистрация
                                </Button>
                            </Link>
                        </div>
                    )}
                </div>
            </div>
        </header>
    );
}
