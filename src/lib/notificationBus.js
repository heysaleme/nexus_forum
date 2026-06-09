/** Lightweight pub/sub for realtime notification events from the global WebSocket. */
const listeners = new Set();

export function onRealtimeNotification(listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
}

export function emitRealtimeNotification(message) {
    listeners.forEach((listener) => {
        try {
            listener(message);
        } catch (err) {
            console.error('[nexus:notification] listener error', err);
        }
    });
}

export function logRealtimeNotification(direction, message) {
    const payload = {
        direction,
        type: message?.type,
        notificationId: message?.data?.id,
        notificationType: message?.data?.type,
        count: message?.count,
        at: new Date().toISOString(),
    };
    console.info('[nexus:notification]', payload);
}
