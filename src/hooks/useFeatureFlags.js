import { useEffect, useState } from 'react';
import { nexusApi } from '@/api/nexusApi';

let cachedFlags = null;
let inflight = null;

async function loadFlags() {
    if (cachedFlags) return cachedFlags;
    if (!inflight) {
        inflight = nexusApi.featureFlags.getPublic()
            .then((flags) => {
                cachedFlags = flags || {};
                return cachedFlags;
            })
            .catch(() => ({}))
            .finally(() => {
                inflight = null;
            });
    }
    return inflight;
}

/** Public feature flags from GET /api/feature-flags (defaults true until loaded). */
export function useFeatureFlags() {
    const [flags, setFlags] = useState(cachedFlags || {
        crosspost: true,
        web_push: true,
        live_ws: true,
    });

    useEffect(() => {
        let active = true;
        loadFlags().then((f) => {
            if (active) setFlags({ crosspost: true, web_push: true, live_ws: true, ...f });
        });
        return () => { active = false; };
    }, []);

    return flags;
}

export function invalidateFeatureFlagsCache() {
    cachedFlags = null;
}
