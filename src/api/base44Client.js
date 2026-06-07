const DB_KEY = 'nexus_forum_local_db';
const SESSION_KEY = 'nexus_forum_session_token';
const RESET_PREFIX = 'nexus_forum_reset_token_';
const OTP_CODE = '123456';

const now = () => new Date().toISOString();

const makeId = (prefix) => `${prefix}_${Math.random().toString(36).slice(2, 10)}`;

const listeners = new Map();

const ACHIEVEMENT_META = {
    first_post: { description: 'Первый пост', tier: 'silver' },
    first_comment: { description: 'Первый комментарий', tier: 'silver' },
    community_builder: { description: 'Создатель сообщества', tier: 'gold' },
    social_butterfly: { description: 'Первый фолловинг', tier: 'bronze' },
};

const notifySubscribers = (entityName, event) => {
    const entityListeners = listeners.get(entityName) || [];
    entityListeners.forEach((listener) => listener(event));
};

const sortRecords = (records, sortSpec) => {
    if (!sortSpec) return records;
    const desc = sortSpec.startsWith('-');
    const field = desc ? sortSpec.slice(1) : sortSpec;

    return [...records].sort((a, b) => {
        const aValue = a?.[field];
        const bValue = b?.[field];

        if (aValue == null && bValue == null) return 0;
        if (aValue == null) return 1;
        if (bValue == null) return -1;

        if (typeof aValue === 'number' && typeof bValue === 'number') {
            return desc ? bValue - aValue : aValue - bValue;
        }

        const aDate = Date.parse(aValue);
        const bDate = Date.parse(bValue);
        if (!Number.isNaN(aDate) && !Number.isNaN(bDate)) {
            return desc ? bDate - aDate : aDate - bDate;
        }

        const aText = String(aValue).toLowerCase();
        const bText = String(bValue).toLowerCase();
        if (aText < bText) return desc ? 1 : -1;
        if (aText > bText) return desc ? -1 : 1;
        return 0;
    });
};

const matchesFilter = (record, filter = {}) => Object.entries(filter).every(([key, expected]) => {
    const actual = record?.[key];
    if (Array.isArray(actual)) return actual.includes(expected);
    return actual === expected;
});

const createSeedData = () => {
    const user1 = {
        id: 'user_demo',
        email: 'amira@example.com',
        password: 'password123',
        full_name: 'Amira',
        username: 'amira',
        avatar_url: '',
        banner_url: '',
        bio: 'Люблю аниме, интерфейсы и аккуратный фронтенд.',
        role: 'admin',
        level: 5,
        xp: 430,
        title: 'Frontend Builder',
        profile_theme: 'sunset',
        followers_count: 12,
        following_count: 5,
        karma: 128,
        allow_dms: true,
        is_private: false,
        is_banned: false,
        created_date: now(),
    };

    const user2 = {
        id: 'user_kai',
        email: 'kai@example.com',
        password: 'password123',
        full_name: 'Kai',
        username: 'kaizer',
        avatar_url: '',
        banner_url: '',
        bio: 'Собираю фанатские сообщества и пишу посты.',
        role: 'user',
        level: 3,
        xp: 240,
        title: 'Community Mod',
        profile_theme: 'ocean',
        followers_count: 3,
        following_count: 8,
        karma: 64,
        allow_dms: true,
        is_private: false,
        is_banned: false,
        created_date: now(),
    };

    const community1 = {
        id: 'community_nexus',
        name: 'Nexus Anime',
        slug: 'nexus-anime',
        description: 'Обсуждаем аниме, мангу и любимые фандомы.',
        category: 'anime',
        avatar_url: '',
        banner_url: '',
        owner_id: user1.id,
        tags: ['anime', 'manga', 'fandom'],
        rules: [{ title: 'Уважение', description: 'Без токсичности и оскорблений.' }],
        is_private: false,
        is_nsfw: false,
        member_count: 14,
        post_count: 2,
        activity_level: 'high',
        created_date: now(),
    };

    const community2 = {
        id: 'community_ui',
        name: 'UI Workshop',
        slug: 'ui-workshop',
        description: 'Разбор интерфейсов, анимаций и продуктового дизайна.',
        category: 'technology',
        avatar_url: '',
        banner_url: '',
        owner_id: user2.id,
        tags: ['design', 'frontend', 'ux'],
        rules: [{ title: 'Конструктив', description: 'Критикуем бережно и по делу.' }],
        is_private: false,
        is_nsfw: false,
        member_count: 9,
        post_count: 1,
        activity_level: 'trending',
        created_date: now(),
    };

    const community3 = {
        id: 'community_rp',
        name: 'Roleplay Hub',
        slug: 'roleplay-hub',
        description: 'Поиск игроков, сюжетов и вселенных для ролевых игр.',
        category: 'roleplay',
        avatar_url: '',
        banner_url: '',
        owner_id: user1.id,
        tags: ['rp', 'stories'],
        rules: [{ title: '18+', description: 'Возрастные ограничения указываем явно.' }],
        is_private: false,
        is_nsfw: false,
        member_count: 21,
        post_count: 1,
        activity_level: 'medium',
        created_date: now(),
    };

    const post1 = {
        id: 'post_welcome',
        title: 'Как вам новый дизайн ленты?',
        content: 'Собрала первый рабочий вариант локального форума. Хочется понять, где интерфейс уже хорош, а где еще сырой.',
        type: 'text',
        author_id: user1.id,
        author_username: user1.full_name,
        author_avatar: user1.avatar_url,
        community_id: community2.id,
        community_name: community2.name,
        community_avatar: community2.avatar_url,
        media_urls: [],
        tags: ['ui', 'feedback'],
        status: 'published',
        score: 18,
        upvotes: 18,
        downvotes: 0,
        views: 74,
        comment_count: 2,
        created_date: new Date(Date.now() - 1000 * 60 * 90).toISOString(),
    };

    const post2 = {
        id: 'post_anime',
        title: 'Топ аниме-сообществ для новичков',
        content: 'Сделала подборку дружелюбных тредов, где комфортно начинать общение и не бояться задавать вопросы.',
        type: 'text',
        author_id: user2.id,
        author_username: user2.full_name,
        author_avatar: user2.avatar_url,
        community_id: community1.id,
        community_name: community1.name,
        community_avatar: community1.avatar_url,
        media_urls: [],
        tags: ['anime', 'guide'],
        status: 'published',
        score: 9,
        upvotes: 10,
        downvotes: 1,
        views: 42,
        comment_count: 1,
        created_date: new Date(Date.now() - 1000 * 60 * 60 * 8).toISOString(),
    };

    const post3 = {
        id: 'post_rp',
        title: 'Ищу игроков для sci-fi RP',
        content: 'Нужны 2-3 человека в мягкую сюжетную космооперу с упором на персонажей.',
        type: 'text',
        author_id: user1.id,
        author_username: user1.full_name,
        author_avatar: user1.avatar_url,
        community_id: community3.id,
        community_name: community3.name,
        community_avatar: community3.avatar_url,
        media_urls: [],
        tags: ['rp', 'sci-fi'],
        status: 'published',
        score: 6,
        upvotes: 7,
        downvotes: 1,
        views: 31,
        comment_count: 0,
        created_date: new Date(Date.now() - 1000 * 60 * 60 * 30).toISOString(),
    };

    return {
        User: [user1, user2],
        Community: [community1, community2, community3],
        CommunityMember: [
            { id: 'member_1', user_id: user1.id, community_id: community1.id, role: 'owner', created_date: now() },
            { id: 'member_2', user_id: user1.id, community_id: community2.id, role: 'member', created_date: now() },
            { id: 'member_3', user_id: user1.id, community_id: community3.id, role: 'owner', created_date: now() },
            { id: 'member_4', user_id: user2.id, community_id: community1.id, role: 'member', created_date: now() },
            { id: 'member_5', user_id: user2.id, community_id: community2.id, role: 'owner', created_date: now() },
        ],
        Post: [post1, post2, post3],
        Comment: [
            {
                id: 'comment_1',
                post_id: post1.id,
                author_id: user2.id,
                author_username: user2.full_name,
                author_avatar: '',
                content: 'Мне нравится структура. Особенно хорошо сработала правая колонка с сообществами.',
                score: 4,
                depth: 0,
                created_date: new Date(Date.now() - 1000 * 60 * 40).toISOString(),
            },
            {
                id: 'comment_2',
                post_id: post1.id,
                author_id: user1.id,
                author_username: user1.full_name,
                author_avatar: '',
                content: 'Спасибо, хочу еще доработать пустые состояния и onboarding.',
                score: 2,
                depth: 1,
                parent_id: 'comment_1',
                created_date: new Date(Date.now() - 1000 * 60 * 20).toISOString(),
            },
            {
                id: 'comment_3',
                post_id: post2.id,
                author_id: user1.id,
                author_username: user1.full_name,
                author_avatar: '',
                content: 'Добавь еще раздел с рекомендациями по жанрам.',
                score: 3,
                depth: 0,
                created_date: new Date(Date.now() - 1000 * 60 * 75).toISOString(),
            },
        ],
        Vote: [],
        SavedPost: [
            {
                id: 'saved_1',
                user_id: user1.id,
                post_id: post2.id,
                post_title: post2.title,
                post_community: post2.community_name,
                created_date: now(),
            },
        ],
        Achievement: [
            { id: 'ach_1', user_id: user1.id, achievement_name: 'community_builder', achievement_description: 'Создатель сообщества', tier: 'gold' },
            { id: 'ach_2', user_id: user1.id, achievement_name: 'first_post', achievement_description: 'Первый пост', tier: 'silver' },
        ],
        UserFollow: [
            { id: 'follow_1', follower_id: user2.id, following_id: user1.id, created_date: now() },
        ],
        Notification: [
            {
                id: 'notif_1',
                user_id: user1.id,
                type: 'reply',
                title: 'Новый ответ',
                body: 'Kai ответил на ваш комментарий.',
                actor_avatar: '',
                is_read: false,
                created_date: new Date(Date.now() - 1000 * 60 * 15).toISOString(),
            },
            {
                id: 'notif_2',
                user_id: user1.id,
                type: 'follow',
                title: 'Новый подписчик',
                body: 'Kai подписался на ваш профиль.',
                actor_avatar: '',
                is_read: true,
                created_date: new Date(Date.now() - 1000 * 60 * 120).toISOString(),
            },
        ],
        Report: [
            { id: 'report_1', reason: 'spam', status: 'pending', post_id: post3.id, created_date: now() },
        ],
        ChatRoom: [
            {
                id: 'chat_1',
                name: 'Amira & Kai',
                type: 'direct',
                participants: [user1.id, user2.id],
                avatar_url: '',
                unread_count: 1,
                last_message: 'Сделала локальный mock-клиент, теперь все открывается без Base44.',
                last_message_at: new Date(Date.now() - 1000 * 60 * 25).toISOString(),
            },
        ],
        Message: [
            {
                id: 'msg_1',
                chat_room_id: 'chat_1',
                sender_id: user2.id,
                sender_username: user2.full_name,
                sender_avatar: '',
                content: 'Какой следующий шаг по проекту?',
                message_type: 'text',
                is_read: true,
                created_date: new Date(Date.now() - 1000 * 60 * 60).toISOString(),
            },
            {
                id: 'msg_2',
                chat_room_id: 'chat_1',
                sender_id: user1.id,
                sender_username: user1.full_name,
                sender_avatar: '',
                content: 'Сделала локальный mock-клиент, теперь все открывается без Base44.',
                message_type: 'text',
                is_read: false,
                created_date: new Date(Date.now() - 1000 * 60 * 25).toISOString(),
            },
        ],
        WikiArticle: [
            {
                id: 'wiki_1',
                title: 'Как оформить сообщество',
                content: 'Подберите тему, правила и понятный onboarding для новичков.',
                community_name: community2.name,
                category: 'guide',
                views: 22,
                created_date: now(),
            },
        ],
    };
};

const readDb = () => {
    const raw = localStorage.getItem(DB_KEY);
    if (raw) return JSON.parse(raw);
    const seed = createSeedData();
    localStorage.setItem(DB_KEY, JSON.stringify(seed));
    if (!localStorage.getItem(SESSION_KEY)) {
        localStorage.setItem(SESSION_KEY, 'token_user_demo');
    }
    return seed;
};

const writeDb = (db) => {
    localStorage.setItem(DB_KEY, JSON.stringify(db));
    return db;
};

const updateDb = (updater) => {
    const db = readDb();
    const next = updater(structuredClone(db));
    return writeDb(next);
};

const recalculateLevel = (user) => {
    user.level = Math.max(1, Math.floor((user.xp || 0) / 100) + 1);
};

const addProgress = (db, userId, { xp = 0, karma = 0 } = {}) => {
    const user = db.User.find((item) => item.id === userId);
    if (!user) return;
    user.xp = Math.max(0, (user.xp || 0) + xp);
    user.karma = Math.max(0, (user.karma || 0) + karma);
    recalculateLevel(user);
};

const ensureAchievement = (db, userId, achievementName) => {
    if (db.Achievement.some((item) => item.user_id === userId && item.achievement_name === achievementName)) {
        return;
    }

    const meta = ACHIEVEMENT_META[achievementName];
    if (!meta) return;

    db.Achievement.push({
        id: makeId('achievement'),
        user_id: userId,
        achievement_name: achievementName,
        achievement_description: meta.description,
        tier: meta.tier,
        created_date: now(),
    });
};

const getCurrentUser = () => {
    const token = localStorage.getItem(SESSION_KEY);
    if (!token) return null;
    const userId = token.replace('token_', '');
    const db = readDb();
    return db.User.find((user) => user.id === userId) || null;
};

const setCurrentUser = (userId) => {
    localStorage.setItem(SESSION_KEY, `token_${userId}`);
};

const createEntityApi = (entityName) => ({
    async list(sortSpec = null, limit = null) {
        const records = sortRecords(readDb()[entityName] || [], sortSpec);
        return limit ? records.slice(0, limit) : records;
    },
    async filter(filter = {}, sortSpec = null, limit = null) {
        const filtered = (readDb()[entityName] || []).filter((record) => matchesFilter(record, filter));
        const sorted = sortRecords(filtered, sortSpec);
        return limit ? sorted.slice(0, limit) : sorted;
    },
    async create(payload) {
        const record = { id: makeId(entityName.toLowerCase()), created_date: now(), ...payload };
        updateDb((db) => {
            db[entityName].push(record);

            if (entityName === 'CommunityMember') {
                const community = db.Community.find((item) => item.id === payload.community_id);
                if (community) community.member_count = (community.member_count || 0) + 1;
            }

            if (entityName === 'Post') {
                const community = db.Community.find((item) => item.id === payload.community_id);
                if (community) community.post_count = (community.post_count || 0) + 1;
                addProgress(db, payload.author_id, { xp: 20, karma: 5 });
                const userPosts = db.Post.filter((item) => item.author_id === payload.author_id);
                if (userPosts.length === 1) {
                    ensureAchievement(db, payload.author_id, 'first_post');
                }
            }

            if (entityName === 'Comment') {
                const post = db.Post.find((item) => item.id === payload.post_id);
                if (post) post.comment_count = (post.comment_count || 0) + 1;
                addProgress(db, payload.author_id, { xp: 8, karma: 2 });
                const userComments = db.Comment.filter((item) => item.author_id === payload.author_id);
                if (userComments.length === 1) {
                    ensureAchievement(db, payload.author_id, 'first_comment');
                }
            }

            if (entityName === 'UserFollow') {
                const follower = db.User.find((item) => item.id === payload.follower_id);
                const following = db.User.find((item) => item.id === payload.following_id);
                if (follower) follower.following_count = (follower.following_count || 0) + 1;
                if (following) following.followers_count = (following.followers_count || 0) + 1;
                addProgress(db, payload.follower_id, { xp: 4, karma: 1 });
                ensureAchievement(db, payload.follower_id, 'social_butterfly');
            }

            if (entityName === 'Message') {
                const room = db.ChatRoom.find((item) => item.id === payload.chat_room_id);
                if (room) {
                    room.last_message = payload.content;
                    room.last_message_at = record.created_date;
                }
            }

            if (entityName === 'Community') {
                addProgress(db, payload.owner_id, { xp: 30, karma: 10 });
                ensureAchievement(db, payload.owner_id, 'community_builder');
            }

            return db;
        });

        notifySubscribers(entityName, { type: 'create', data: record });
        return record;
    },
    async update(id, payload) {
        let updatedRecord = null;
        updateDb((db) => {
            db[entityName] = db[entityName].map((record) => {
                if (record.id !== id) return record;
                updatedRecord = { ...record, ...payload };
                return updatedRecord;
            });
            return db;
        });
        if (!updatedRecord) throw new Error(`${entityName} not found`);
        notifySubscribers(entityName, { type: 'update', data: updatedRecord });
        return updatedRecord;
    },
    async delete(id) {
        let deletedRecord = null;
        updateDb((db) => {
            const all = db[entityName];
            deletedRecord = all.find((record) => record.id === id) || null;
            db[entityName] = all.filter((record) => record.id !== id);

            if (deletedRecord && entityName === 'CommunityMember') {
                const community = db.Community.find((item) => item.id === deletedRecord.community_id);
                if (community) community.member_count = Math.max(0, (community.member_count || 1) - 1);
            }

            if (deletedRecord && entityName === 'UserFollow') {
                const follower = db.User.find((item) => item.id === deletedRecord.follower_id);
                const following = db.User.find((item) => item.id === deletedRecord.following_id);
                if (follower) follower.following_count = Math.max(0, (follower.following_count || 1) - 1);
                if (following) following.followers_count = Math.max(0, (following.followers_count || 1) - 1);
            }

            return db;
        });
        notifySubscribers(entityName, { type: 'delete', data: deletedRecord });
        return { success: true };
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
        const user = getCurrentUser();
        if (!user) {
            const error = new Error('Not authenticated');
            error.status = 401;
            throw error;
        }
        return user;
    },
    async loginViaEmailPassword(email, password) {
        const db = readDb();
        const user = db.User.find((item) => item.email.toLowerCase() === email.toLowerCase() && item.password === password);
        if (!user) throw new Error('Invalid email or password');
        setCurrentUser(user.id);
        return { access_token: `token_${user.id}` };
    },
    async register({ email, password }) {
        const db = readDb();
        const exists = db.User.some((item) => item.email.toLowerCase() === email.toLowerCase());
        if (exists) throw new Error('Email already registered');
        localStorage.setItem(`${RESET_PREFIX}pending_registration`, JSON.stringify({ email, password }));
        return { success: true, otp_required: true };
    },
    async verifyOtp({ email, otpCode }) {
        if (otpCode !== OTP_CODE) throw new Error('Use demo OTP code 123456');
        const pending = JSON.parse(localStorage.getItem(`${RESET_PREFIX}pending_registration`) || 'null');
        if (!pending || pending.email !== email) throw new Error('No pending registration for this email');

        const user = {
            id: makeId('user'),
            email,
            password: pending.password,
            full_name: email.split('@')[0],
            username: email.split('@')[0],
            avatar_url: '',
            banner_url: '',
            bio: '',
            role: 'user',
            level: 1,
            xp: 0,
            title: '',
            profile_theme: 'default',
            followers_count: 0,
            following_count: 0,
            karma: 0,
            allow_dms: true,
            is_private: false,
            is_banned: false,
            created_date: now(),
        };

        updateDb((db) => {
            db.User.push(user);
            return db;
        });
        localStorage.removeItem(`${RESET_PREFIX}pending_registration`);
        setCurrentUser(user.id);
        return { access_token: `token_${user.id}` };
    },
    async resendOtp() {
        return { success: true };
    },
    setToken(token) {
        localStorage.setItem(SESSION_KEY, token);
    },
    loginWithProvider(_provider, redirectPath = '/') {
        setCurrentUser('user_demo');
        window.location.href = redirectPath;
    },
    async updateMe(profile) {
        const user = await auth.me();
        const updated = await base44.entities.User.update(user.id, {
            ...profile,
            full_name: profile.username || user.full_name,
            username: profile.username || user.username,
        });
        return updated;
    },
    logout(redirectPath) {
        localStorage.removeItem(SESSION_KEY);
        if (redirectPath) window.location.href = typeof redirectPath === 'string' ? redirectPath : '/';
    },
    redirectToLogin() {
        window.location.href = '/login';
    },
    async resetPasswordRequest(email) {
        const db = readDb();
        const user = db.User.find((item) => item.email.toLowerCase() === email.toLowerCase());
        if (user) {
            localStorage.setItem(`${RESET_PREFIX}${email}`, user.id);
        }
        return { success: true };
    },
    async resetPassword({ resetToken, newPassword }) {
        const userId = resetToken;
        const db = readDb();
        const user = db.User.find((item) => item.id === userId);
        if (!user) throw new Error('Invalid reset token');
        await base44.entities.User.update(userId, { password: newPassword });
        return { success: true };
    },
};

const base44 = {
    auth,
    integrations: {
        Core: {
            async UploadFile({ file }) {
                return { file_url: URL.createObjectURL(file) };
            },
        },
    },
    entities: {
        User: createEntityApi('User'),
        Community: createEntityApi('Community'),
        CommunityMember: createEntityApi('CommunityMember'),
        Post: createEntityApi('Post'),
        Comment: createEntityApi('Comment'),
        Vote: createEntityApi('Vote'),
        SavedPost: createEntityApi('SavedPost'),
        Achievement: createEntityApi('Achievement'),
        UserFollow: createEntityApi('UserFollow'),
        Notification: createEntityApi('Notification'),
        Report: createEntityApi('Report'),
        ChatRoom: createEntityApi('ChatRoom'),
        Message: createEntityApi('Message'),
        WikiArticle: createEntityApi('WikiArticle'),
    },
};

if (typeof window !== 'undefined') {
    window.__NEXUS_DB__ = {
        read: () => readDb(),
        reset: () => {
            localStorage.removeItem(DB_KEY);
            localStorage.removeItem(SESSION_KEY);
        },
    };
}

export { base44 };
