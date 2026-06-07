const BASE_URL = 'http://localhost:8080/api';
const SESSION_KEY = 'nexus_forum_session_token';

const listeners = new Map();

const notifySubscribers = (entityName, event) => {
    const entityListeners = listeners.get(entityName) || [];
    entityListeners.forEach((listener) => listener(event));
};

const getToken = () => {
    return localStorage.getItem(SESSION_KEY);
};

const setToken = (token) => {
    localStorage.setItem(SESSION_KEY, token);
};

const request = async (path, options = {}) => {
    const token = getToken();
    const headers = {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
        ...options.headers,
    };

    const response = await fetch(`${BASE_URL}${path}`, {
        ...options,
        headers,
    });

    if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        const err = new Error(errData.error || `HTTP error ${response.status}`);
        err.status = response.status;
        throw err;
    }

    // 204 No Content doesn't have JSON body
    if (response.status === 204) {
        return { success: true };
    }

    return response.json();
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
        try {
            return await request('/auth/me');
        } catch (e) {
            if (e.status === 401) {
                localStorage.removeItem(SESSION_KEY);
            }
            throw e;
        }
    },
    async loginViaEmailPassword(email, password) {
        const res = await request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
        if (res.access_token) {
            setToken(res.access_token);
        }
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
        if (res.access_token) {
            setToken(res.access_token);
        }
        return res;
    },
    async resendOtp() {
        return { success: true };
    },
    setToken(token) {
        setToken(token);
    },
    loginWithProvider(provider, redirectPath = '/') {
        // Mock provider login - redirect directly
        window.location.href = redirectPath;
    },
    async updateMe(profile) {
        return request('/auth/me', {
            method: 'PUT',
            body: JSON.stringify(profile),
        });
    },
    logout(redirectPath) {
        localStorage.removeItem(SESSION_KEY);
        if (redirectPath) {
            window.location.href = typeof redirectPath === 'string' ? redirectPath : '/';
        }
    },
    redirectToLogin() {
        window.location.href = '/login';
    },
    async resetPasswordRequest(email) {
        return { success: true };
    },
    async resetPassword({ resetToken, newPassword }) {
        return { success: true };
    },
};

const nexusApi = {
    auth,
    Search: {
        async query(q) {
            return request(`/search?q=${encodeURIComponent(q)}`);
        }
    },
    integrations: {
        Core: {
            async UploadFile({ file }) {
                return new Promise((resolve, reject) => {
                    const reader = new FileReader();
                    reader.onloadend = () => resolve({ file_url: reader.result });
                    reader.onerror = reject;
                    reader.readAsDataURL(file);
                });
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
                return Array.isArray(res) ? res.map(parsePost) : res;
            }
        },
        Comment: {
            ...createEntityApi('Comment', '/comments'),
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
                        const followers = await request(`/users/${filter.following_id}/followers`);
                        const found = (followers || []).find(f => f.id === filter.follower_id || f.follower_id === filter.follower_id);
                        return found ? [{ id: filter.following_id, ...found }] : [];
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
            }
        },
        Notification: {
            ...createEntityApi('Notification', '/notifications'),
            async readAll() {
                return request('/notifications/read', { method: 'POST' });
            }
        },
        Report: {
            async list() {
                try {
                    return await request('/moderation/reports');
                } catch {
                    return [
                        { id: 1, status: 'pending', reason: 'spam', target_type: 'post', description: 'Этот пост содержит спам-ссылки', reporter_username: 'kaizer' },
                        { id: 2, status: 'pending', reason: 'harassment', target_type: 'comment', description: 'Оскорбление пользователей в комментариях', reporter_username: 'moduser' },
                    ];
                }
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
            async banUser(id, reason) {
                return request(`/moderation/users/${id}/ban`, { method: 'POST', body: JSON.stringify({ reason }) });
            },
            async unbanUser(id, reason) {
                return request(`/moderation/users/${id}/unban`, { method: 'POST', body: JSON.stringify({ reason }) });
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
                    body: JSON.stringify({ content: payload.content }),
                });
                notifySubscribers('Message', { type: 'create', data: record });
                return record;
            },
            async delete(id) {
                const res = await request(`/messages/${id}`, {
                    method: 'DELETE',
                });
                notifySubscribers('Message', { type: 'delete', data: { id } });
                return res;
            },
        },
    },
};

export { nexusApi };
