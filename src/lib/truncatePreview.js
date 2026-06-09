/**
 * Truncate post text for feed previews without breaking words when possible.
 */
export function truncatePreview(text, maxLength) {
    if (!text || maxLength <= 0) return '';
    const normalized = String(text).trim();
    if (normalized.length <= maxLength) return normalized;

    const slice = normalized.slice(0, maxLength);
    const lastSpace = slice.lastIndexOf(' ');
    const cut = lastSpace > maxLength * 0.6 ? slice.slice(0, lastSpace) : slice;

    return `${cut.trimEnd()}...`;
}
