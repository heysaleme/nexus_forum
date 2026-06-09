const SW_LOG = '[nexus-sw]';

self.addEventListener('install', (event) => {
    console.info(SW_LOG, 'install');
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    console.info(SW_LOG, 'activate');
    event.waitUntil(self.clients.claim());
});

function broadcastToClients(message) {
    return self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
        clientList.forEach((client) => client.postMessage(message));
    });
}

self.addEventListener('push', (event) => {
    console.info(SW_LOG, 'push event received', {
        hasData: Boolean(event.data),
    });

    let data = { title: 'Nexus Forum', body: 'Новое уведомление', url: '/notifications' };
    try {
        if (event.data) {
            data = { ...data, ...event.data.json() };
        }
    } catch {
        if (event.data) {
            data.body = event.data.text();
        }
    }

    const options = {
        body: data.body,
        icon: '/favicon.ico',
        badge: '/favicon.ico',
        tag: data.type || 'nexus-notification',
        data: { url: data.url || '/notifications', type: data.type },
    };

    event.waitUntil(
        self.registration.showNotification(data.title || 'Nexus Forum', options)
            .then(() => {
                console.info(SW_LOG, 'showNotification resolved', { title: data.title, body: data.body });
                return broadcastToClients({
                    type: 'push_received',
                    title: data.title,
                    body: data.body,
                    at: new Date().toISOString(),
                });
            })
            .catch((err) => {
                console.error(SW_LOG, 'showNotification failed', err);
                return broadcastToClients({
                    type: 'push_error',
                    error: String(err),
                    at: new Date().toISOString(),
                });
            })
    );
});

self.addEventListener('notificationclick', (event) => {
    console.info(SW_LOG, 'notificationclick', event.notification?.data);
    event.notification.close();
    const targetUrl = event.notification.data?.url || '/notifications';
    event.waitUntil(
        self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
            for (const client of clientList) {
                if (client.url.includes(targetUrl) && 'focus' in client) {
                    return client.focus();
                }
            }
            if (self.clients.openWindow) {
                return self.clients.openWindow(targetUrl);
            }
            return undefined;
        })
    );
});
