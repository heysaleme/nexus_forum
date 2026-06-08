import React, { useState } from "react";
import { Link } from "react-router-dom";
import { nexusApi } from '@/api/nexusApi';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Mail, ArrowLeft, Loader2 } from "lucide-react";
import AuthLayout from "@/components/AuthLayout";

export default function ForgotPassword() {
    const [email, setEmail] = useState("");
    const [loading, setLoading] = useState(false);
    const [sent, setSent] = useState(false);
    const [devResetLink, setDevResetLink] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setLoading(true);
        setDevResetLink('');
        try {
            const res = await nexusApi.auth.resetPasswordRequest(email);
            if (res?.reset_token) {
                setDevResetLink(`/reset-password?token=${encodeURIComponent(res.reset_token)}`);
            }
            setSent(true);
        } catch (err) {
            setSent(true);
        } finally {
            setLoading(false);
        }
    };

    return (
        <AuthLayout
            icon={Mail}
            title="Reset password"
            subtitle="We'll send you a link to reset it"
            footer={
                <Link to="/login" className="text-primary font-medium hover:underline">
                    <ArrowLeft className="w-3 h-3 inline mr-1" />Back to log in
                </Link>
            }
        >
            {sent ? (
                <div className="space-y-3 text-sm text-foreground text-center">
                    <p>
                        If an account exists with that email, you'll receive a password reset link shortly.
                    </p>
                    {devResetLink && (
                        <p className="text-xs text-muted-foreground break-all">
                            Dev reset link:{' '}
                            <Link to={devResetLink} className="text-primary hover:underline">
                                {devResetLink}
                            </Link>
                        </p>
                    )}
                </div>
            ) : (
                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="space-y-2">
                        <Label htmlFor="email">Email address</Label>
                        <div className="relative">
                            <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" aria-hidden="true" />
                            <Input
                                id="email"
                                type="email"
                                autoComplete="email"
                                autoFocus
                                placeholder="you@example.com"
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                className="pl-10 h-12"
                                required
                            />
                        </div>
                    </div>
                    <Button type="submit" className="w-full h-12 font-medium" disabled={loading}>
                        {loading ? (
                            <>
                                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                                Sending...
                            </>
                        ) : (
                            "Send reset link"
                        )}
                    </Button>
                </form>
            )}
        </AuthLayout>
    );
}
