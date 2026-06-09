import { nexusApi } from '@/api/nexusApi';

const SW_URL = '/sw.js';

if (typeof navigator !== 'undefined' && 'serviceWorker' in navigator) {
    navigator.serviceWorker.addEventListener('message', (event) => {
        if (event.data?.type === 'push_received') {
            console.info('[nexus:push] service worker delivered notification to OS', event.data);
        } else if (event.data?.type === 'push_error') {
            console.error('[nexus:push] service worker showNotification failed', event.data);
        }
    });
}

function urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);
    for (let i = 0; i < rawData.length; i += 1) {
        outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
}

export function isPushSupported() {
    return typeof window !== 'undefined'
        && 'serviceWorker' in navigator
        && 'PushManager' in window
        && typeof Notification !== 'undefined';
}

export function getNotificationPermission() {
    if (typeof Notification === 'undefined') return 'unsupported';
    return Notification.permission;
}

/** Register SW and wait until active (required before PushManager.subscribe). */
export async function ensureServiceWorkerRegistration() {
    if (!('serviceWorker' in navigator)) {
        throw new Error('Service workers are not supported in this browser');
    }

    let registration = await navigator.serviceWorker.getRegistration('/');
    if (!registration) {
        registration = await navigator.serviceWorker.register(SW_URL, { scope: '/' });
    }

    if (registration.installing) {
        await new Promise((resolve, reject) => {
            const worker = registration.installing;
            const timeout = setTimeout(() => reject(new Error('Service worker install timeout')), 15000);
            worker.addEventListener('statechange', () => {
                if (worker.state === 'activated' || worker.state === 'redundant') {
                    clearTimeout(timeout);
                    resolve();
                }
            });
        });
    }

    await navigator.serviceWorker.ready;
    return registration;
}

export async function getLocalPushSubscription() {
    if (!isPushSupported()) return null;
    const reg = await ensureServiceWorkerRegistration();
    return reg.pushManager.getSubscription();
}

export async function subscribeToPush() {
    if (!isPushSupported()) {
        throw new Error('Push notifications are not supported in this browser');
    }

    const permission = await Notification.requestPermission();
    if (permission !== 'granted') {
        throw new Error('Notification permission was not granted');
    }

    const registration = await ensureServiceWorkerRegistration();
    const { public_key: vapidKey } = await nexusApi.push.getVapidPublicKey();
    if (!vapidKey) {
        throw new Error('Push is not configured on the server (missing VAPID keys)');
    }

    let subscription = await registration.pushManager.getSubscription();
    if (!subscription) {
        subscription = await registration.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey: urlBase64ToUint8Array(vapidKey),
        });
    }

    const json = subscription.toJSON();
    await nexusApi.push.subscribe({
        endpoint: json.endpoint,
        p256dh: json.keys?.p256dh,
        auth: json.keys?.auth,
    });

    return subscription;
}

export async function unsubscribeFromPush() {
    if (!isPushSupported()) return;
    const registration = await ensureServiceWorkerRegistration();
    const subscription = await registration.pushManager.getSubscription();
    if (subscription) {
        await nexusApi.push.unsubscribe(subscription.endpoint);
        await subscription.unsubscribe();
    }
}
