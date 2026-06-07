import { Link, useParams } from 'react-router-dom';
import { BookOpen, ArrowLeft } from 'lucide-react';
import EmptyState from '@/components/ui/EmptyState';
import { Button } from '@/components/ui/button';

export default function WikiPage() {
    const { id } = useParams();

    return (
        <div className="max-w-4xl mx-auto px-4 py-6">
            <div className="mb-4">
                <Link to="/" className="inline-flex">
                    <Button variant="ghost" size="sm" className="rounded-xl gap-2">
                        <ArrowLeft className="w-4 h-4" />
                        Назад
                    </Button>
                </Link>
            </div>

            <div className="nexus-card p-6">
                <EmptyState
                    icon={BookOpen}
                    title={id ? 'Статья вики пока недоступна' : 'Раздел вики в разработке'}
                    description={id
                        ? 'Маршрут для статьи уже подключен, но сама страница еще не реализована.'
                        : 'Базовый маршрут настроен. Теперь приложение может открываться без ошибки отсутствующего файла.'}
                    action={
                        <Link to="/communities">
                            <Button className="nexus-gradient border-0 text-white rounded-xl shadow-nexus">
                                Перейти к сообществам
                            </Button>
                        </Link>
                    }
                />
            </div>
        </div>
    );
}
