let pending = null;

export function requestCaptchaChallenge() {
    return new Promise((resolve, reject) => {
        pending = { resolve, reject };
        window.dispatchEvent(new CustomEvent('nexus:captcha-request'));
    });
}

export function submitCaptchaToken(token) {
    if (pending) {
        pending.resolve(token);
        pending = null;
    }
}

export function cancelCaptchaChallenge(error) {
    if (pending) {
        pending.reject(error || new Error('CAPTCHA cancelled'));
        pending = null;
    }
}
