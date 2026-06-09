import React from 'react'
import ReactDOM from 'react-dom/client'
import App from '@/App.jsx'
import '@/index.css'
import { ensureServiceWorkerRegistration, isPushSupported } from '@/lib/pushNotifications'

if (isPushSupported()) {
    ensureServiceWorkerRegistration().catch((err) => {
        console.warn('Service worker registration failed:', err);
    });
}

ReactDOM.createRoot(document.getElementById('root')).render(
    <App />
)
