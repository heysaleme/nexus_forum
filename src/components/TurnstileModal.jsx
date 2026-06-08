import { useEffect, useRef, useState } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { submitCaptchaToken, cancelCaptchaChallenge } from '@/lib/captcha';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';

const TURNSTILE_SCRIPT = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';

function loadTurnstileScript() {
    if (window.turnstile) return Promise.resolve();
    if (document.querySelector(`script[src^="${TURNSTILE_SCRIPT}"]`)) {
        return new Promise((resolve) => {
            const check = () => (window.turnstile ? resolve() : setTimeout(check, 50));
            check();
        });
    }
    return new Promise((resolve, reject) => {
        const script = document.createElement('script');
        script.src = TURNSTILE_SCRIPT;
        script.async = true;
        script.onload = () => resolve();
        script.onerror = () => reject(new Error('Failed to load Turnstile'));
        document.head.appendChild(script);
    });
}

export default function TurnstileModal() {
    const [open, setOpen] = useState(false);
    const [siteKey, setSiteKey] = useState('');
    const containerRef = useRef(null);
    const widgetIdRef = useRef(null);

    useEffect(() => {
        nexusApi.auth.getOAuthConfig()
            .then((cfg) => setSiteKey(cfg.turnstile_site_key || import.meta.env.VITE_TURNSTILE_SITE_KEY || ''))
            .catch(() => setSiteKey(import.meta.env.VITE_TURNSTILE_SITE_KEY || ''));
    }, []);

    useEffect(() => {
        const onRequest = () => {
            if (!siteKey) {
                cancelCaptchaChallenge(new Error('CAPTCHA не настроена на сервере'));
                return;
            }
            setOpen(true);
        };
        window.addEventListener('nexus:captcha-request', onRequest);
        return () => window.removeEventListener('nexus:captcha-request', onRequest);
    }, [siteKey]);

    useEffect(() => {
        if (!open || !siteKey || !containerRef.current) return undefined;

        let cancelled = false;

        loadTurnstileScript()
            .then(() => {
                if (cancelled || !containerRef.current) return;
                containerRef.current.innerHTML = '';
                widgetIdRef.current = window.turnstile.render(containerRef.current, {
                    sitekey: siteKey,
                    callback: (token) => {
                        submitCaptchaToken(token);
                        setOpen(false);
                    },
                    'error-callback': () => {
                        cancelCaptchaChallenge(new Error('Ошибка CAPTCHA'));
                        setOpen(false);
                    },
                    'expired-callback': () => {
                        cancelCaptchaChallenge(new Error('CAPTCHA истекла'));
                        setOpen(false);
                    },
                });
            })
            .catch((err) => {
                cancelCaptchaChallenge(err);
                setOpen(false);
            });

        return () => {
            cancelled = true;
            if (widgetIdRef.current != null && window.turnstile) {
                try {
                    window.turnstile.remove(widgetIdRef.current);
                } catch {
                    // ignore cleanup errors
                }
                widgetIdRef.current = null;
            }
        };
    }, [open, siteKey]);

    const handleOpenChange = (next) => {
        if (!next && open) {
            cancelCaptchaChallenge(new Error('CAPTCHA отменена'));
        }
        setOpen(next);
    };

    return (
        <Dialog open={open} onOpenChange={handleOpenChange}>
            <DialogContent className="sm:max-w-md rounded-2xl">
                <DialogHeader>
                    <DialogTitle>Подтвердите, что вы не робот</DialogTitle>
                    <DialogDescription>
                        Пройдите проверку CAPTCHA, чтобы продолжить публикацию.
                    </DialogDescription>
                </DialogHeader>
                <div ref={containerRef} className="flex justify-center min-h-[65px]" />
            </DialogContent>
        </Dialog>
    );
}
