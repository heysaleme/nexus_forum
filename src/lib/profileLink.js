/** Decode logged-in user id from JWT (works before /auth/me finishes). */
export function getUserIdFromToken() {
    try {
        const token = localStorage.getItem('nexus_forum_session_token');
        if (!token) return null;
        const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
        return payload.user_id ?? payload.UserID ?? null;
    } catch {
        return null;
    }
}

/** Profile route: own profile uses /profile, others use /user/:id */
export function profilePath(userId, currentUserId) {
    if (!userId) return '/profile';
    const selfId = currentUserId ?? getUserIdFromToken();
    if (isOwnProfile(userId, selfId)) return '/profile';
    return `/user/${userId}`;
}

export function isOwnProfile(profileId, currentUserId) {
    if (!profileId || !currentUserId) return !profileId;
    return Number(profileId) === Number(currentUserId);
}

export function sameUserId(a, b) {
    if (a == null || b == null) return false;
    return Number(a) === Number(b);
}

export function displayName(user) {
    if (!user) return 'Пользователь';
    return user.username || user.full_name || user.email || 'Пользователь';
}
