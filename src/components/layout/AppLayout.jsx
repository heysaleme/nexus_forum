import { useEffect } from 'react';
import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import BottomNav from './BottomNav';
import TopBar from './TopBar';
import { useAuth } from '@/lib/AuthContext';
import { ChatLayoutProvider, useChatLayout } from '@/lib/ChatLayoutContext';

function AppLayoutInner() {
    const { user } = useAuth();
    const { mobileChatOpen } = useChatLayout();
    const hideChrome = mobileChatOpen;

    useEffect(() => {
        document.documentElement.classList.add('app-shell');
        return () => document.documentElement.classList.remove('app-shell');
    }, []);

    return (
        <div className={`bg-background flex max-w-[100vw] fixed inset-0 overflow-hidden ${hideChrome ? 'z-[90]' : ''}`}>
            {!hideChrome && <Sidebar user={user} />}
            <div className={`flex-1 flex flex-col min-w-0 min-h-0 ${!hideChrome ? 'md:ml-64' : ''}`}>
                {!hideChrome && <TopBar user={user} />}
                <main className={`flex-1 min-h-0 overflow-y-auto overflow-x-hidden ${hideChrome ? 'pb-0' : 'pb-20 md:pb-0'}`}>
                    <Outlet />
                </main>
            </div>
            {!hideChrome && <BottomNav />}
        </div>
    );
}

export default function AppLayout() {
    return (
        <ChatLayoutProvider>
            <AppLayoutInner />
        </ChatLayoutProvider>
    );
}