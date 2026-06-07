import React, { createContext, useState, useContext, useEffect } from 'react';
import { base44 } from '@/api/base44Client';

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    const [isAuthenticated, setIsAuthenticated] = useState(false);
    const [isLoadingAuth, setIsLoadingAuth] = useState(true);
    const [isLoadingPublicSettings, setIsLoadingPublicSettings] = useState(false);
    const [authError, setAuthError] = useState(null);
    const [authChecked, setAuthChecked] = useState(false);
    const [appPublicSettings] = useState({ id: 'local-nexus', public_settings: { auth_required: false } });

    const checkUserAuth = async () => {
        setIsLoadingAuth(true);
        try {
            const currentUser = await base44.auth.me();
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

    const logout = (shouldRedirect = true) => {
        setUser(null);
        setIsAuthenticated(false);
        base44.auth.logout(shouldRedirect ? '/' : undefined);
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
