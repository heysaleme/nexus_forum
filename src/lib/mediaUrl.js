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

/** Store relative upload paths in the database (local backend). */
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

/** Strip presigned query params; keep stable object reference for MinIO/S3 and local paths. */
export function canonicalStorageUrl(url) {
    if (!url) return '';
    if (url.startsWith('/')) return url;
    try {
        const parsed = new URL(url);
        return `${parsed.origin}${parsed.pathname}`;
    } catch {
        return url;
    }
}

/** Pick the canonical reference from an upload API response. */
export function storageReferenceFromUpload(data) {
    if (!data) return '';
    if (data.storage_url) return data.storage_url;
    if (data.file_url) return canonicalStorageUrl(data.file_url);
    if (data.url) return canonicalStorageUrl(data.url);
    return '';
}
