import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import BottomNav from './BottomNav';
import TopBar from './TopBar';
import { useAuth } from '@/lib/AuthContext';

export default function AppLayout() {
    const { user } = useAuth();

    return (
        <div className="min-h-screen bg-background flex">
            <Sidebar user={user} />
            <div className="flex-1 flex flex-col min-w-0">
                <TopBar user={user} />
                <main className="flex-1 pb-20 md:pb-0">
                    <Outlet />
                </main>
            </div>
            <BottomNav />
        </div>
    );
}