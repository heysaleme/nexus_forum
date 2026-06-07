import { Toaster } from "@/components/ui/toaster"
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClientInstance } from '@/lib/query-client'
import { BrowserRouter as Router, Route, Routes, Navigate } from 'react-router-dom';
import PageNotFound from './lib/PageNotFound';
import { AuthProvider, useAuth } from '@/lib/AuthContext';
import UserNotRegisteredError from '@/components/UserNotRegisteredError';

// Layout
import AppLayout from '@/components/layout/AppLayout';

// Pages
import Home from '@/pages/Home';
import Communities from '@/pages/Communities';
import CommunityPage from '@/pages/CommunityPage';
import CreatePost from '@/pages/CreatePost';
import CreateCommunity from '@/pages/CreateCommunity';
import PostPage from '@/pages/PostPage';
import Profile from '@/pages/Profile';
import Notifications from '@/pages/Notifications';
import Chats from '@/pages/Chats';
import Search from '@/pages/Search';
import Settings from '@/pages/Settings';
import AdminPanel from '@/pages/AdminPanel';
import WikiPage from '@/pages/WikiPage';

const AuthenticatedApp = () => {
    const { isLoadingAuth, isLoadingPublicSettings, authError, navigateToLogin } = useAuth();

    if (isLoadingPublicSettings || isLoadingAuth) {
        return (
            <div className="fixed inset-0 flex items-center justify-center bg-background">
                <div className="flex flex-col items-center gap-4">
                    <div className="w-12 h-12 nexus-gradient rounded-3xl flex items-center justify-center shadow-nexus animate-bounce-soft">
                        <span className="text-white text-xl">⚡</span>
                    </div>
                    <div className="w-8 h-8 border-4 border-primary/20 border-t-primary rounded-full animate-spin"></div>
                </div>
            </div>
        );
    }

    if (authError) {
        if (authError.type === 'user_not_registered') {
            return <UserNotRegisteredError />;
        } else if (authError.type === 'auth_required') {
            navigateToLogin();
            return null;
        }
    }

    return (
        <Routes>
            <Route element={<AppLayout />}>
                <Route path="/" element={<Home />} />
                <Route path="/communities" element={<Communities />} />
                <Route path="/community/:id" element={<CommunityPage />} />
                <Route path="/create" element={<CreatePost />} />
                <Route path="/create-community" element={<CreateCommunity />} />
                <Route path="/post/:id" element={<PostPage />} />
                <Route path="/profile" element={<Profile />} />
                <Route path="/user/:id" element={<Profile />} />
                <Route path="/notifications" element={<Notifications />} />
                <Route path="/chats" element={<Chats />} />
                <Route path="/search" element={<Search />} />
                <Route path="/settings" element={<Settings />} />
                <Route path="/admin" element={<AdminPanel />} />
                <Route path="/wiki" element={<WikiPage />} />
                <Route path="/wiki/:id" element={<WikiPage />} />
            </Route>
            <Route path="*" element={<PageNotFound />} />
        </Routes>
    );
};

function App() {
    return (
        <AuthProvider>
            <QueryClientProvider client={queryClientInstance}>
                <Router>
                    <AuthenticatedApp />
                </Router>
                <Toaster />
            </QueryClientProvider>
        </AuthProvider>
    );
}

export default App;
