import { requestCaptchaChallenge } from '@/lib/captcha';

const BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';
const SESSION_KEY = 'nexus_forum_session_token';
const REFRESH_KEY = 'nexus_forum_refresh_token';

const listeners = new Map();

const notifySubscribers = (entityName, event) => {
    const entityListeners = listeners.get(entityName) || [];
    entityListeners.forEach((listener) => listener(event));
};

const getToken = () => localStorage.getItem(SESSION_KEY);
const getRefreshToken = () => localStorage.getItem(REFRESH_KEY);

const setToken = (token) => {
    if (token) localStorage.setItem(SESSION_KEY, token);
};

const persistSession = (res) => {
    if (res?.access_token) setToken(res.access_token);
    if (res?.refresh_token) localStorage.setItem(REFRESH_KEY, res.refresh_token);
};

const clearSession = () => {
    localStorage.removeItem(SESSION_KEY);
    localStorage.removeItem(REFRESH_KEY);
};

/** Decode JWT payload without verification (client-side session id only). */
const getSessionIdFromToken = () => {
    try {
        const token = getToken();
        if (!token) return null;
        const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
        return payload.sid || null;
    } catch {
        return null;
    }
};

let refreshPromise = null;

const refreshAccessToken = async () => {
    const refresh = getRefreshToken();
    if (!refresh) throw new Error('No refresh token');

    if (!refreshPromise) {
        refreshPromise = fetch(`${BASE_URL}/auth/refresh`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: refresh }),
        })
            .then(async (response) => {
                if (!response.ok) {
                    const errData = await response.json().catch(() => ({}));
                    const err = new Error(errData.error || 'Session expired');
                    err.status = response.status;
                    throw err;
                }
                const data = await response.json();
                persistSession(data);
                return data.access_token;
            })
            .finally(() => {
                refreshPromise = null;
            });
    }
    return refreshPromise;
};

const request = async (path, options = {}, retried = false) => {
    const token = getToken();
    const headers = {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...options.headers,
    };

    const response = await fetch(`${BASE_URL}${path}`, {
        ...options,
        headers,
    });

    if (response.status === 401 && !retried && getRefreshToken() && !path.startsWith('/auth/')) {
        try {
            await refreshAccessToken();
            return request(path, options, true);
        } catch {
            clearSession();
            if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
                window.location.href = '/login?reason=session_expired';
            }
        }
    }

    if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        const err = new Error(errData.error || `HTTP error ${response.status}`);
        err.status = response.status;
        if (response.status === 401 && /session revoked/i.test(errData.error || '')) {
            clearSession();
            if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
                window.location.href = '/login?reason=session_revoked';
            }
        }
        throw err;
    }

    if (response.status === 204) {
        return { success: true };
    }

    return response.json();
};

const requestWithCaptcha = async (path, options = {}) => {
    try {
        return await request(path, options);
    } catch (err) {
        if (err.status === 403 && /captcha/i.test(err.message || '')) {
            const turnstileToken = await requestCaptchaChallenge();
            return request(path, {
                ...options,
                headers: {
                    ...options.headers,
                    'X-Turnstile-Token': turnstileToken,
                },
            });
        }
        throw err;
    }
};

const buildQueryString = (filter, sortSpec, limit) => {
    const params = new URLSearchParams();
    if (filter) {
        Object.entries(filter).forEach(([key, value]) => {
            if (value !== undefined && value !== null) {
                params.append(key, value);
            }
        });
    }
    if (sortSpec) {
        params.append('sort', sortSpec);
    }
    if (limit) {
        params.append('limit', limit);
    }
    const str = params.toString();
    return str ? `?${str}` : '';
};

const parsePost = (post) => {
    if (!post) return post;

    // Tags
    if (typeof post.tags === 'string') {
        try {
            post.tags = JSON.parse(post.tags);
        } catch {
            post.tags = [];
        }
    }

    if (!Array.isArray(post.tags)) {
        post.tags = [];
    }

    // Media
    if (typeof post.media_urls === 'string') {
        try {
            post.media_urls = JSON.parse(post.media_urls);
        } catch {
            post.media_urls = [];
        }
    }

    if (!Array.isArray(post.media_urls)) {
        post.media_urls = [];
    }

    // Poll options
    if (typeof post.poll_options === 'string') {
        try {
            post.poll_options = JSON.parse(post.poll_options);
        } catch {
            post.poll_options = [];
        }
    }

    if (!Array.isArray(post.poll_options)) {
        post.poll_options = [];
    }

    return post;
};

const parseCommunity = (community) => {
    if (!community) return community;
    if (typeof community.rules === 'string') {
        try {
            community.rules = JSON.parse(community.rules);
        } catch {
            community.rules = [];
        }
    }
    return community;
};

const createEntityApi = (entityName, endpoint) => ({
    async list(sortSpec = null, limit = null) {
        const query = buildQueryString(null, sortSpec, limit);
        const res = await request(`${endpoint}${query}`);
        if (entityName === 'Post' && Array.isArray(res)) return res.map(parsePost);
        if (entityName === 'Community' && Array.isArray(res)) return res.map(parseCommunity);
        return res;
    },
    async filter(filter = {}, sortSpec = null, limit = null) {
        // If filter has ID, we fetch single record
        if (filter.id) {
            try {
                const record = await request(`${endpoint}/${filter.id}`);
                if (entityName === 'Post') return [parsePost(record)];
                if (entityName === 'Community') return [parseCommunity(record)];
                return [record];
            } catch (e) {
                if (e.status === 404) return [];
                throw e;
            }
        }
        const query = buildQueryString(filter, sortSpec, limit);
        const res = await request(`${endpoint}${query}`);
        if (entityName === 'Post' && Array.isArray(res)) return res.map(parsePost);
        if (entityName === 'Community' && Array.isArray(res)) return res.map(parseCommunity);
        return res;
    },
    async create(payload) {
        const record = await request(endpoint, {
            method: 'POST',
            body: JSON.stringify(payload),
        });
        let resRecord = record;
        if (entityName === 'Post') resRecord = parsePost(record);
        if (entityName === 'Community') resRecord = parseCommunity(record);
        notifySubscribers(entityName, { type: 'create', data: resRecord });
        return resRecord;
    },
    async update(id, payload) {
        const record = await request(`${endpoint}/${id}`, {
            method: 'PUT',
            body: JSON.stringify(payload),
        });
        let resRecord = record;
        if (entityName === 'Post') resRecord = parsePost(record);
        if (entityName === 'Community') resRecord = parseCommunity(record);
        notifySubscribers(entityName, { type: 'update', data: resRecord });
        return resRecord;
    },
    async delete(id) {
        const res = await request(`${endpoint}/${id}`, {
            method: 'DELETE',
        });
        notifySubscribers(entityName, { type: 'delete', data: { id } });
        return res;
    },
    subscribe(listener) {
        const entityListeners = listeners.get(entityName) || [];
        listeners.set(entityName, [...entityListeners, listener]);
        return () => {
            const current = listeners.get(entityName) || [];
            listeners.set(entityName, current.filter((item) => item !== listener));
        };
    },
});

const auth = {
    async me() {
        const token = getToken();
        if (!token) {
            const error = new Error('Not authenticated');
            error.status = 401;
            throw error;
        }
        return request('/auth/me');
    },
    async loginViaEmailPassword(email, password) {
        const res = await request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
        persistSession(res);
        return res;
    },
    async register({ email, password }) {
        return request('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
    },
    async verifyOtp({ email, otpCode }) {
        const res = await request('/auth/verify-otp', {
            method: 'POST',
            body: JSON.stringify({ email, otpCode }),
        });
        persistSession(res);
        return res;
    },
    async confirmEmail(token) {
        const res = await request(`/auth/confirm-email?token=${encodeURIComponent(token)}`, {
            method: 'GET',
        });
        persistSession(res);
        return res;
    },
    async resendOtp(email) {
        return request('/auth/resend-otp', {
            method: 'POST',
            body: JSON.stringify({ email }),
        });
    },
    setToken(token) {
        setToken(token);
    },
    getRefreshToken() {
        return getRefreshToken();
    },
    persistSession,
    clearSession,
    loginWithProvider(provider, redirectPath = '/') {
        // Legacy stub — kept for compatibility; use googleOAuthUrl() instead
        console.warn('loginWithProvider is deprecated. Use googleOAuthUrl() for real OAuth.');
        window.location.href = redirectPath;
    },
    async getOAuthConfig() {
        // Returns { google_enabled: bool, apple_enabled: bool }
        return request('/auth/oauth/config', { method: 'GET' });
    },
    async googleOAuthUrl() {
        // Returns { url: string, state: string } — redirect user to url
        return request('/auth/oauth/google', { method: 'GET' });
    },
    async googleOAuthCallback(code, state) {
        const res = await request('/auth/oauth/google/callback', {
            method: 'POST',
            body: JSON.stringify({ code, state }),
        });
        persistSession(res);
        return res;
    },
    async githubOAuthUrl() {
        return request('/auth/oauth/github', { method: 'GET' });
    },
    async githubOAuthCallback(code, state) {
        const res = await request('/auth/oauth/github/callback', {
            method: 'POST',
            body: JSON.stringify({ code, state }),
        });
        persistSession(res);
        return res;
    },
    async refreshSession() {
        const access = await refreshAccessToken();
        return { access_token: access };
    },
    async updateMe(profile) {
        return request('/auth/me', {
            method: 'PUT',
            body: JSON.stringify(profile),
        });
    },
    async listSessions() {
        return request('/auth/sessions');
    },
    async revokeSession(sessionId) {
        const currentSid = getSessionIdFromToken();
        const res = await request(`/auth/sessions/${sessionId}`, { method: 'DELETE' });
        if (currentSid && Number(currentSid) === Number(sessionId)) {
            clearSession();
            window.location.href = '/login?reason=session_revoked';
        }
        return res;
    },
    getSessionIdFromToken,
    async revokeOtherSessions(keepSessionId) {
        const sid = keepSessionId || getSessionIdFromToken();
        return request('/auth/sessions/revoke-others', {
            method: 'POST',
            body: JSON.stringify({ keep_session_id: sid ? Number(sid) : 0 }),
        });
    },
    async logout(redirectPath) {
        const refresh = getRefreshToken();
        try {
            if (refresh) {
                await fetch(`${BASE_URL}/auth/logout`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ refresh_token: refresh }),
                });
            }
        } catch {
            // still clear local session
        }
        clearSession();
        if (redirectPath) {
            window.location.href = typeof redirectPath === 'string' ? redirectPath : '/';
        }
    },
    redirectToLogin() {
        window.location.href = '/login';
    },
    async resetPasswordRequest(email) {
        return request('/auth/forgot-password', {
            method: 'POST',
            body: JSON.stringify({ email }),
        });
    },
    async resetPassword({ resetToken, newPassword }) {
        return request('/auth/reset-password', {
            method: 'POST',
            body: JSON.stringify({ resetToken, newPassword }),
        });
    },
    async changePassword({ old_password, new_password }) {
        return request('/auth/change-password', {
            method: 'POST',
            body: JSON.stringify({ old_password, new_password }),
        });
    },
};

const nexusApi = {
    BASE_URL,
    auth,
    analytics: {
        async getDashboard() {
            return request('/analytics/dashboard');
        },
        async getActivity(days = 7) {
            const res = await request(`/analytics/activity?days=${days}`);
            return res?.activity || [];
        },
        async getReportReasons() {
            const res = await request('/analytics/reports');
            return res?.report_reasons || [];
        },
        async getRetention() {
            const res = await request('/analytics/retention');
            return res?.retention || {};
        },
        async getEngagement() {
            return request('/analytics/engagement');
        },
    },
    feed: {
        async list({ sort = 'hot', limit = 50 } = {}) {
            const params = new URLSearchParams({ sort, limit: String(limit) });
            const res = await request(`/posts?${params}`);
            return Array.isArray(res) ? res.map(parsePost) : [];
        },
        async following({ sort = 'new', limit = 50 } = {}) {
            const params = new URLSearchParams({ sort, limit: String(limit) });
            const res = await request(`/posts/following?${params}`);
            return Array.isArray(res) ? res.map(parsePost) : [];
        },
        async followingCommunities({ sort = 'new', limit = 50 } = {}) {
            const params = new URLSearchParams({ sort, limit: String(limit) });
            const res = await request(`/posts/following-communities?${params}`);
            return Array.isArray(res) ? res.map(parsePost) : [];
        },
    },
    Search: {
        async query(q) {
            const data = await request(`/search?q=${encodeURIComponent(q)}`);
            if (data?.posts && Array.isArray(data.posts)) {
                data.posts = data.posts.map(parsePost);
            }
            return data;
        }
    },
    integrations: {
        Core: {
            async UploadFile({ file, category = 'chat/attachments' }) {
                const formData = new FormData();
                formData.append('file', file);
                formData.append('category', category);

                const token = getToken();
                const response = await fetch(`${BASE_URL}/upload`, {
                    method: 'POST',
                    headers: token ? { Authorization: `Bearer ${token}` } : {},
                    body: formData,
                });

                if (!response.ok) {
                    const errData = await response.json().catch(() => ({}));
                    const err = new Error(errData.error || `Upload failed (${response.status})`);
                    err.status = response.status;
                    throw err;
                }

                const data = await response.json();
                let url = data.file_url || data.url;
                if (url && url.startsWith('/')) {
                    const origin = BASE_URL.replace(/\/api\/?$/, '');
                    url = `${origin}${url}`;
                }
                return { file_url: url, mime_type: data.mime_type, filename: data.filename };
            },
        },
    },
    entities: {
        User: createEntityApi('User', '/users'),
        Community: createEntityApi('Community', '/communities'),
        CommunityMember: {
            ...createEntityApi('CommunityMember', '/communities'),
            async filter(filter = {}) {
                // If user is checking their community memberships
                if (filter.user_id) {
                    const memberships = await request(`/communities/memberships?user_id=${filter.user_id}`);
                    return memberships;
                }
                // If checking members of a community
                if (filter.community_id) {
                    const members = await request(`/communities/${filter.community_id}/members`);
                    return members;
                }
                return [];
            },
            async create(payload) {
                const res = await request(`/communities/${payload.community_id}/join`, {
                    method: 'POST',
                });
                notifySubscribers('CommunityMember', { type: 'create', data: payload });
                return res;
            },
            async delete(id) {
                // To leave community, we expect payload with community_id
                // We'll call /communities/:id/leave
                // In frontend, `CommunityPage.jsx` does: CommunityMember.delete(myMember.id)
                // However, since we use relational model, we can query community_id and leave
                // Let's support leaving by extracting membership
                const res = await request(`/communities/${id}/leave`, {
                    method: 'POST',
                });
                notifySubscribers('CommunityMember', { type: 'delete', data: { id } });
                return res;
            }
        },
        Post: {
            ...createEntityApi('Post', '/posts'),
            async create(payload) {
                const record = await requestWithCaptcha('/posts', {
                    method: 'POST',
                    body: JSON.stringify(payload),
                });
                const resRecord = parsePost(record);
                notifySubscribers('Post', { type: 'create', data: resRecord });
                return resRecord;
            },
            async list(sortSpec = null, limit = null) {
                const query = buildQueryString(null, sortSpec, limit);
                const res = await request(`/posts${query}`);
                return Array.isArray(res) ? res.map(parsePost) : res;
            },
            async filter(filter = {}, sortSpec = null, limit = null) {
                // Map frontend author_id and community_id filters
                const params = {};
                if (filter.author_id) params.author_id = filter.author_id;
                if (filter.community_id) params.community_id = filter.community_id;
                if (filter.status) params.status = filter.status;
                if (filter.id) {
                    try {
                        const record = await request(`/posts/${filter.id}`);
                        return [parsePost(record)];
                    } catch (e) {
                        return [];
                    }
                }
                const query = buildQueryString(params, sortSpec, limit);
                const res = await request(`/posts${query}`);
                let posts = Array.isArray(res) ? res.map(parsePost) : res;
                if (Array.isArray(posts) && filter.status) {
                    posts = posts.filter((p) => p.status === filter.status);
                }
                return posts;
            }
        },
        Comment: {
            ...createEntityApi('Comment', '/comments'),
            async create(payload) {
                const record = await requestWithCaptcha('/comments', {
                    method: 'POST',
                    body: JSON.stringify(payload),
                });
                notifySubscribers('Comment', { type: 'create', data: record });
                return record;
            },
            async filter(filter = {}) {
                if (filter.post_id) {
                    return request(`/comments?post_id=${filter.post_id}`);
                }
                return [];
            }
        },
        Vote: {
            async create(payload) {
                // In frontend: Vote.create({ target_id: post.id, target_type: 'post', value })
                const endpoint = payload.target_type === 'post' ? `/posts/${payload.target_id}/vote` : `/comments/${payload.target_id}/vote`;
                return request(endpoint, {
                    method: 'POST',
                    body: JSON.stringify({ value: payload.value }),
                });
            }
        },
        PollVote: {
            async create(postId, optionIndex) {
                return request(`/posts/${postId}/poll`, {
                    method: 'POST',
                    body: JSON.stringify({
                        option_index: optionIndex
                    }),
                });
            }
        },
        SavedPost: {
            async filter(filter = {}) {
                if (filter.user_id) {
                    return request('/users/saved');
                }
                return [];
            },
            async create(payload) {
                return request(`/posts/${payload.post_id}/save`, {
                    method: 'POST',
                });
            },
            async delete(id) {
                // id here is the post id
                return request(`/posts/${id}/unsave`, {
                    method: 'POST',
                });
            }
        },
        Achievement: {
            async filter(filter = {}) {
                // Mock achievements dynamically based on level/posts
                const user = await auth.me().catch(() => null);
                if (!user) return [];
                const achievements = [];
                if (user.level >= 3) {
                    achievements.push({ id: 'ach_1', user_id: user.id, achievement_name: 'community_builder', achievement_description: 'Создатель сообщества', tier: 'gold' });
                }
                if (user.xp > 20) {
                    achievements.push({ id: 'ach_2', user_id: user.id, achievement_name: 'first_post', achievement_description: 'Первый пост', tier: 'silver' });
                }
                return achievements;
            }
        },
        UserFollow: {
            async filter(filter = {}) {
                // Check if follower_id follows following_id
                if (filter.follower_id && filter.following_id) {
                    try {
                        const res = await request(`/users/${filter.following_id}/follow-status`);
                        if (res && res.status && res.status !== 'none') {
                            return [{ id: filter.following_id, status: res.status }];
                        }
                        return [];
                    } catch {
                        return [];
                    }
                }
                // Get all users that follower_id is following
                if (filter.follower_id) {
                    try {
                        return await request(`/users/${filter.follower_id}/following`);
                    } catch {
                        return [];
                    }
                }
                return [];
            },
            async create(payload) {
                return request(`/users/${payload.following_id}/follow`, {
                    method: 'POST',
                });
            },
            async delete(id) {
                // id is the following user ID
                return request(`/users/${id}/unfollow`, {
                    method: 'POST',
                });
            },
            async getPendingRequests() {
                return request('/users/follow-requests');
            },
            async acceptRequest(followerId) {
                return request(`/users/follow-requests/${followerId}/accept`, {
                    method: 'POST',
                });
            },
            async rejectRequest(followerId) {
                return request(`/users/follow-requests/${followerId}/reject`, {
                    method: 'POST',
                });
            }
        },
        Notification: {
            ...createEntityApi('Notification', '/notifications'),
            async list() {
                return request('/notifications');
            },
            async readAll() {
                return request('/notifications/read', { method: 'POST' });
            },
            async markRead(id) {
                return request(`/notifications/${id}/read`, { method: 'POST' });
            },
        },
        Report: {
            async list() {
                return request('/moderation/reports');
            },
            async create(payload) {
                return request('/reports', {
                    method: 'POST',
                    body: JSON.stringify(payload),
                });
            },
            async update(id, payload) {
                return request(`/moderation/reports/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify(payload),
                });
            }
        },
        Moderation: {
            async banUser(id, reason = 'moderation action') {
                return request(`/moderation/users/${id}/ban`, { method: 'POST', body: JSON.stringify({ reason }) });
            },
            async unbanUser(id, reason = 'moderation action') {
                return request(`/moderation/users/${id}/unban`, { method: 'POST', body: JSON.stringify({ reason }) });
            },
            async shadowBanUser(id, reason = 'moderation action') {
                return request(`/moderation/users/${id}/shadow-ban`, { method: 'POST', body: JSON.stringify({ reason }) });
            },
            async unshadowBanUser(id, reason = 'moderation action') {
                return request(`/moderation/users/${id}/unshadow-ban`, { method: 'POST', body: JSON.stringify({ reason }) });
            },
            async removePost(id, reason) {
                return request(`/moderation/posts/${id}/remove`, { method: 'POST', body: JSON.stringify({ reason }) });
            },
            async removeComment(id, reason) {
                return request(`/moderation/comments/${id}/remove`, { method: 'POST', body: JSON.stringify({ reason }) });
            }
        },
        ChatRoom: createEntityApi('ChatRoom', '/chats'),
        Message: {
            ...createEntityApi('Message', '/messages'),
            async filter(filter = {}) {
                if (filter.chat_room_id) {
                    return request(`/chats/${filter.chat_room_id}/messages`);
                }
                return [];
            },
            async create(payload) {
                // ChatRoom message creation
                const record = await request(`/chats/${payload.chat_room_id}/messages`, {
                    method: 'POST',
                    body: JSON.stringify({
                        content: payload.content,
                        attachment_url: payload.attachment_url,
                        attachment_type: payload.attachment_type,
                    }),
                });
                notifySubscribers('Message', { type: 'create', data: record });
                return record;
            },
            async update(id, content) {
                const record = await request(`/messages/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify({ content }),
                });
                notifySubscribers('Message', { type: 'update', data: record });
                return record;
            },
            async delete(id, deleteType = 'me') {
                const res = await request(`/messages/${id}?type=${deleteType}`, {
                    method: 'DELETE',
                });
                notifySubscribers('Message', { type: 'delete', data: { id, deleteType } });
                return res;
            },
        },
    },
};

export { nexusApi };
