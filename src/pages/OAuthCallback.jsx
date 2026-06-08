import React, { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { nexusApi } from "@/api/nexusApi";
import { Loader2 } from "lucide-react";

/**
 * OAuthCallback — Google OAuth redirect landing page.
 *
 * Google redirects here with ?code=...&state=... after the user approves access.
 * This component reads those params, sends them to the backend for validation
 * and token exchange, stores the returned JWT, then navigates to the home page.
 *
 * Route: /auth/callback/google
 */
export default function OAuthCallback({ provider = "google" }) {
    const navigate = useNavigate();
    const [error, setError] = useState("");

    useEffect(() => {
        const params = new URLSearchParams(window.location.search);
        const code = params.get("code");
        const state = params.get("state");

        if (!code || !state) {
            setError("Missing OAuth parameters. Please try logging in again.");
            return;
        }

        const callback = provider === "github"
            ? nexusApi.auth.githubOAuthCallback(code, state)
            : nexusApi.auth.googleOAuthCallback(code, state);

        callback
            .then(() => {
                // Token is already stored by googleOAuthCallback — navigate home
                navigate("/", { replace: true });
            })
            .catch((err) => {
                setError(err.message || "Authentication failed. Please try again.");
            });
    }, [navigate, provider]);

    if (error) {
        return (
            <div className="min-h-screen flex flex-col items-center justify-center gap-4 px-4">
                <div className="max-w-sm w-full p-6 rounded-xl bg-destructive/10 border border-destructive/20 text-center">
                    <p className="text-destructive font-medium mb-2">Login failed</p>
                    <p className="text-sm text-muted-foreground mb-4">{error}</p>
                    <a
                        href="/login"
                        className="text-sm text-primary hover:underline"
                    >
                        Back to login
                    </a>
                </div>
            </div>
        );
    }

    return (
        <div className="min-h-screen flex flex-col items-center justify-center gap-3">
            <Loader2 className="w-8 h-8 animate-spin text-primary" />
            <p className="text-sm text-muted-foreground">Completing sign in…</p>
        </div>
    );
}
