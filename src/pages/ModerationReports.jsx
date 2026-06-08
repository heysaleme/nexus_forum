import { useState, useEffect } from 'react';
import { nexusApi } from '@/api/nexusApi';
import { useAuth } from '@/lib/AuthContext';
import { useNavigate } from 'react-router-dom';
import LoadingSpinner from '@/components/ui/LoadingSpinner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Shield, AlertTriangle, CheckCircle, ChevronDown, ChevronUp, User, MessageCircle, FileText } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useToast } from '@/components/ui/use-toast';

export default function ModerationReports() {
    const { user } = useAuth();
    const navigate = useNavigate();
    const { toast } = useToast();
    const [reports, setReports] = useState([]);
    const [loading, setLoading] = useState(true);
    const [filterStatus, setFilterStatus] = useState('all');
    const [filterType, setFilterType] = useState('all');
    
    // Track expanded reports
    const [expandedReportId, setExpandedReportId] = useState(null);
    const [modResponses, setModResponses] = useState({}); // id -> string
    const [submittingId, setSubmittingId] = useState(null);

    useEffect(() => {
        if (!user || (user.role !== 'admin' && user.role !== 'moderator')) {
            navigate('/');
            return;
        }
        loadReports();
    }, [user]);

    const loadReports = async () => {
        setLoading(true);
        try {
            const data = await nexusApi.entities.Report.list();
            setReports(data || []);
        } catch (err) {
            toast({ title: 'Не удалось загрузить список жалоб', variant: 'destructive' });
        }
        setLoading(false);
    };

    const handleAction = async (reportId, actionStatus) => {
        const responseText = modResponses[reportId] || '';
        if (actionStatus === 'resolved' && !responseText.trim()) {
            toast({ title: 'Введите ответ модератора перед разрешением жалобы', variant: 'destructive' });
            return;
        }
        setSubmittingId(reportId);
        try {
            await nexusApi.entities.Report.update(reportId, {
                status: actionStatus,
                moderator_response: responseText.trim(),
            });
            toast({ title: actionStatus === 'resolved' ? '✅ Жалоба успешно разрешена' : '❌ Жалоба отклонена' });
            loadReports();
        } catch (err) {
            toast({ title: 'Не удалось обработать жалобу', variant: 'destructive' });
        }
        setSubmittingId(null);
    };

    const toggleExpand = (id) => {
        setExpandedReportId(expandedReportId === id ? null : id);
    };

    const handleResponseChange = (id, text) => {
        setModResponses(prev => ({ ...prev, [id]: text }));
    };

    if (loading) return <LoadingSpinner size="lg" className="py-32" />;

    const filteredReports = reports.filter(r => {
        const matchesStatus = filterStatus === 'all' || r.status === filterStatus;
        const matchesType = filterType === 'all' || r.target_type === filterType;
        return matchesStatus && matchesType;
    });

    const getTargetIcon = (type) => {
        switch (type) {
            case 'post': return <FileText className="w-4 h-4 text-blue-500" />;
            case 'comment': return <MessageCircle className="w-4 h-4 text-green-500" />;
            case 'user': return <User className="w-4 h-4 text-purple-500" />;
            default: return <AlertTriangle className="w-4 h-4" />;
        }
    };

    const getStatusBadge = (status) => {
        switch (status) {
            case 'pending':
                return <Badge className="bg-orange-100 text-orange-700 border-0">Ожидает</Badge>;
            case 'resolved':
                return <Badge className="bg-green-100 text-green-700 border-0">Разрешена</Badge>;
            case 'rejected':
            case 'dismissed':
                return <Badge className="bg-red-100 text-red-700 border-0">Отклонена</Badge>;
            default:
                return <Badge className="bg-muted text-muted-foreground border-0">{status}</Badge>;
        }
    };

    return (
        <div className="max-w-4xl mx-auto px-4 py-4">
            {/* Header */}
            <div className="flex items-center gap-2 mb-5">
                <div className="w-9 h-9 bg-primary/10 rounded-xl flex items-center justify-center">
                    <Shield className="w-5 h-5 text-primary" />
                </div>
                <div>
                    <h1 className="text-xl font-display font-black">Очередь модерации</h1>
                    <p className="text-xs text-muted-foreground">Обработка жалоб пользователей на публикации, комментарии и профили</p>
                </div>
            </div>

            {/* Filters */}
            <div className="nexus-card p-4 mb-4 flex flex-col sm:flex-row gap-3">
                <div className="flex-1">
                    <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Статус</Label>
                    <Select value={filterStatus} onValueChange={setFilterStatus}>
                        <SelectTrigger className="rounded-xl border-border/50 h-9 text-xs">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">Все статусы</SelectItem>
                            <SelectItem value="pending">Ожидающие</SelectItem>
                            <SelectItem value="resolved">Разрешенные</SelectItem>
                            <SelectItem value="rejected">Отклоненные</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
                <div className="flex-1">
                    <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">Тип контента</Label>
                    <Select value={filterType} onValueChange={setFilterType}>
                        <SelectTrigger className="rounded-xl border-border/50 h-9 text-xs">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="all">Все типы</SelectItem>
                            <SelectItem value="post">Посты</SelectItem>
                            <SelectItem value="comment">Комментарии</SelectItem>
                            <SelectItem value="user">Профили</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
            </div>

            {/* Reports List */}
            <div className="space-y-3">
                {filteredReports.length === 0 ? (
                    <div className="nexus-card p-8 text-center">
                        <CheckCircle className="w-12 h-12 text-green-500 mx-auto mb-2" />
                        <p className="text-sm font-bold text-foreground">Нет жалоб по выбранным фильтрам</p>
                        <p className="text-xs text-muted-foreground">Отличная работа по поддержанию порядка!</p>
                    </div>
                ) : (
                    filteredReports.map(report => {
                        const isExpanded = expandedReportId === report.id;
                        return (
                            <motion.div
                                key={report.id}
                                layout
                                className={`nexus-card overflow-hidden border ${isExpanded ? 'border-primary/50' : 'border-border/30'} transition-all`}
                            >
                                <div
                                    onClick={() => toggleExpand(report.id)}
                                    className="p-4 flex items-center justify-between gap-3 cursor-pointer hover:bg-muted/10 transition-colors"
                                >
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2 flex-wrap mb-1.5">
                                            {getStatusBadge(report.status)}
                                            <Badge variant="outline" className="text-[10px] gap-1 px-2 py-0.5 rounded-md h-5">
                                                {getTargetIcon(report.target_type)}
                                                <span className="capitalize">{report.target_type}</span>
                                            </Badge>
                                            <span className="text-[10px] text-muted-foreground">ID цели: {report.target_id}</span>
                                        </div>
                                        <p className="text-sm font-bold truncate">Причина: {report.reason}</p>
                                        {report.description && (
                                            <p className="text-xs text-muted-foreground line-clamp-1 mt-0.5">{report.description}</p>
                                        )}
                                        <div className="flex items-center gap-1.5 mt-1 text-[10px] text-muted-foreground">
                                            <span>Отправитель: @{report.reporter_username}</span>
                                        </div>
                                    </div>
                                    <div className="flex-shrink-0">
                                        {isExpanded ? <ChevronUp className="w-4 h-4 text-muted-foreground" /> : <ChevronDown className="w-4 h-4 text-muted-foreground" />}
                                    </div>
                                </div>

                                <AnimatePresence>
                                    {isExpanded && (
                                        <motion.div
                                            initial={{ height: 0, opacity: 0 }}
                                            animate={{ height: 'auto', opacity: 1 }}
                                            exit={{ height: 0, opacity: 0 }}
                                            className="border-t border-border/30 bg-muted/10"
                                        >
                                            <div className="p-4 space-y-4">
                                                {/* Details */}
                                                <div className="space-y-2">
                                                    <div>
                                                        <Label className="text-xs font-semibold text-muted-foreground">Описание жалобы</Label>
                                                        <p className="text-xs text-foreground bg-muted/40 p-2.5 rounded-xl border border-border/20 mt-1 whitespace-pre-wrap leading-relaxed">
                                                            {report.description || 'Описание не предоставлено.'}
                                                        </p>
                                                    </div>
                                                    
                                                    {report.moderator_response && (
                                                        <div>
                                                            <Label className="text-xs font-semibold text-muted-foreground">Ответ модератора</Label>
                                                            <p className="text-xs text-foreground bg-primary/5 p-2.5 rounded-xl border border-primary/10 mt-1 whitespace-pre-wrap leading-relaxed">
                                                                {report.moderator_response}
                                                            </p>
                                                        </div>
                                                    )}
                                                </div>

                                                {/* Action form (only for pending reports) */}
                                                {report.status === 'pending' && (
                                                    <div className="space-y-3 pt-2 border-t border-border/20">
                                                        <div>
                                                            <Label className="text-xs font-semibold text-muted-foreground mb-1 block">Ответ модератора (будет отправлен пользователю)</Label>
                                                            <Textarea
                                                                value={modResponses[report.id] || ''}
                                                                onChange={e => handleResponseChange(report.id, e.target.value)}
                                                                placeholder="Введите обоснование решения..."
                                                                className="rounded-xl border-border/50 text-xs min-h-16 resize-none"
                                                                maxLength={300}
                                                            />
                                                        </div>
                                                        <div className="flex gap-2">
                                                            <Button
                                                                onClick={() => handleAction(report.id, 'resolved')}
                                                                disabled={submittingId === report.id || !(modResponses[report.id] || '').trim()}
                                                                className="flex-1 h-8 rounded-xl bg-green-600 hover:bg-green-700 text-white text-xs font-bold border-0 shadow-sm"
                                                            >
                                                                Принять и наказать
                                                            </Button>
                                                            <Button
                                                                onClick={() => handleAction(report.id, 'rejected')}
                                                                disabled={submittingId === report.id}
                                                                variant="outline"
                                                                className="flex-1 h-8 rounded-xl text-xs font-bold border-border"
                                                            >
                                                                Отклонить
                                                            </Button>
                                                        </div>
                                                    </div>
                                                )}
                                            </div>
                                        </motion.div>
                                    )}
                                </AnimatePresence>
                            </motion.div>
                        );
                    })
                )}
            </div>
        </div>
    );
}
