import { nexusApi } from '@/api/nexusApi';

/** Resolve upload path or absolute URL for display. */
export function mediaUrl(path) {
    if (!path) return '';
    if (path.startsWith('http://') || path.startsWith('https://') || path.startsWith('data:')) {
        return path;
    }
    const origin = nexusApi.BASE_URL.replace(/\/api\/?$/, '');
    return path.startsWith('/') ? `${origin}${path}` : `${origin}/${path}`;
}

/** Store relative upload paths in the database. */
export function toRelativeUploadUrl(url) {
    if (!url) return '';
    if (url.startsWith('/')) return url;
    try {
        const parsed = new URL(url);
        return parsed.pathname;
    } catch {
        return url;
    }
}
