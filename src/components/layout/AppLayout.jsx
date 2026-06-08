import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import BottomNav from './BottomNav';
import TopBar from './TopBar';
import { useAuth } from '@/lib/AuthContext';

export default function AppLayout() {
    const { user } = useAuth();

    return (
        <div className="min-h-screen bg-background flex overflow-x-hidden max-w-[100vw]">
            <Sidebar user={user} />
            <div className="flex-1 flex flex-col min-w-0 overflow-x-hidden">
                <TopBar user={user} />
                <main className="flex-1 pb-20 md:pb-0 overflow-x-hidden">
                    <Outlet />
                </main>
            </div>
            <BottomNav />
        </div>
    );
}