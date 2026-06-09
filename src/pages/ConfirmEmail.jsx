import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { nexusApi } from '@/api/nexusApi';
import AuthLayout from '@/components/AuthLayout';
import { Button } from '@/components/ui/button';
import { Mail, Loader2, CheckCircle2 } from 'lucide-react';

export default function ConfirmEmail() {
    const [searchParams] = useSearchParams();
    const token = searchParams.get('token');
    const [status, setStatus] = useState(token ? 'loading' : 'missing');
    const [error, setError] = useState('');

    useEffect(() => {
        if (!token) return;
        (async () => {
            try {
                await nexusApi.auth.confirmEmail(token);
                setStatus('success');
                setTimeout(() => {
                    window.location.href = '/';
                }, 1500);
            } catch (err) {
                setError(err.message || 'Ссылка недействительна или устарела');
                setStatus('error');
            }
        })();
    }, [token]);

    if (status === 'loading') {
        return (
            <AuthLayout icon={Mail} title="Подтверждение email" subtitle="Проверяем ссылку...">
                <div className="flex justify-center py-8">
                    <Loader2 className="w-8 h-8 animate-spin text-primary" />
                </div>
            </AuthLayout>
        );
    }

    if (status === 'success') {
        return (
            <AuthLayout icon={CheckCircle2} title="Email подтверждён" subtitle="Перенаправляем в приложение...">
                <Button asChild className="w-full h-12">
                    <Link to="/">На главную</Link>
                </Button>
            </AuthLayout>
        );
    }

    return (
        <AuthLayout icon={Mail} title="Подтверждение email" subtitle={status === 'missing' ? 'В ссылке нет токена подтверждения' : error}>
            <Button asChild className="w-full h-12">
                <Link to="/register">Зарегистрироваться снова</Link>
            </Button>
        </AuthLayout>
    );
}
