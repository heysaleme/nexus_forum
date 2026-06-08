import React, { createContext, useState, useContext, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [isLoadingAuth, setIsLoadingAuth] = useState(true);
    const [isLoadingPublicSettings, setIsLoadingPublicSettings] = useState(false);
    const [authError, setAuthError] = useState(null);
    const [authChecked, setAuthChecked] = useState(false);
    const [appPublicSettings] = useState({ id: 'local-nexus', public_settings: { auth_required: false } });

    // Auth modal state for guests trying restricted actions
    const [authModalOpen, setAuthModalOpen] = useState(false);
    const [authModalMsg, setAuthModalMsg] = useState('');

    const triggerAuthModal = (message) => {
        setAuthModalMsg(message || 'Для этого действия необходимо войти в аккаунт или зарегистрироваться.');
        setAuthModalOpen(true);
    };

    const checkUserAuth = async () => {
        setIsLoadingAuth(true);
        try {
            const currentUser = await nexusApi.auth.me();
            setUser(currentUser);
            setIsAuthenticated(true);
            setAuthError(null);
        } catch {
            setUser(null);
            setIsAuthenticated(false);
            setAuthError(null);
        } finally {
            setIsLoadingAuth(false);
            setAuthChecked(true);
        }
    };

    useEffect(() => {
        checkUserAuth();
    }, []);

    const checkAppState = async () => {
        await checkUserAuth();
    };

    const logout = async (shouldRedirect = true) => {
        setUser(null);
        setIsAuthenticated(false);
        await nexusApi.auth.logout(shouldRedirect ? '/' : undefined);
    };

    const navigateToLogin = () => {
        window.location.href = '/login';
    };

    return (
        <AuthContext.Provider
            value={{
                user,
                isAuthenticated,
                isLoadingAuth,
                isLoadingPublicSettings,
                authError,
                appPublicSettings,
                authChecked,
                logout,
                navigateToLogin,
                checkUserAuth,
                checkAppState,
                authModalOpen,
                setAuthModalOpen,
                authModalMsg,
                triggerAuthModal,
            }}
        >
            {children}
        </AuthContext.Provider>
    );
};

export const useAuth = () => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
};
