/** Max upload sizes (MB) — must match backend/internal/service/upload_service.go */
export const UPLOAD_LIMITS_MB = {
    'profile/avatars': 5,
    'profile/banners': 8,
    'community/avatars': 5,
    'community/banners': 8,
    'posts/images': 10,
    'posts/videos': 50,
    'chat/attachments': 10,
};

export function validateFileSize(file, maxSizeMB) {
    if (file.size > maxSizeMB * 1024 * 1024) {
        throw new Error(`Файл не должен превышать ${maxSizeMB} МБ`);
    }
}

export function limitLabelForCategory(category) {
    const mb = UPLOAD_LIMITS_MB[category];
    return mb ? `до ${mb} МБ` : '';
}
