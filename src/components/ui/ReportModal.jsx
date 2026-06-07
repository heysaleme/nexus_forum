import { useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useToast } from '@/components/ui/use-toast';
import { nexusApi } from '@/api/nexusApi';

const REPORT_REASONS = [
    { id: 'spam', label: 'Спам' },
    { id: 'harassment', label: 'Оскорбление / Преследование' },
    { id: 'hate_speech', label: 'Язык ненависти' },
    { id: 'misinformation', label: 'Дезинформация' },
    { id: 'nsfw', label: 'NSFW без пометки' },
    { id: 'copyright', label: 'Нарушение авторских прав' },
    { id: 'other', label: 'Другое' },
];

export default function ReportModal({ open, onClose, targetId, targetType, currentUser }) {
    const { toast } = useToast();
    const [reason, setReason] = useState('');
    const [description, setDescription] = useState('');
    const [submitting, setSubmitting] = useState(false);

    const handleSubmit = async () => {
        if (!reason) {
            toast({ title: 'Выберите причину жалобы', variant: 'destructive' });
            return;
        }
        setSubmitting(true);
        try {
            await nexusApi.entities.Report.create({
                reporter_id: currentUser.id,
                reporter_username: currentUser.full_name || currentUser.username,
                target_id: targetId,
                target_type: targetType,
                reason,
                description: description.trim(),
                status: 'pending',
            });
            toast({ title: '✅ Жалоба отправлена. Спасибо!' });
            setReason('');
            setDescription('');
            onClose();
        } catch (err) {
            toast({ title: 'Не удалось отправить жалобу', variant: 'destructive' });
        }
        setSubmitting(false);
    };

    return (
        <Dialog open={open} onOpenChange={onClose}>
            <DialogContent className="sm:max-w-md rounded-2xl p-5 bg-card border border-border">
                <DialogHeader>
                    <DialogTitle className="font-display font-black text-base">
                        🚩 Пожаловаться
                    </DialogTitle>
                </DialogHeader>

                <div className="space-y-4 mt-2">
                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">
                            Причина жалобы
                        </Label>
                        <Select value={reason} onValueChange={setReason}>
                            <SelectTrigger className="rounded-xl border-border/50 h-10 text-sm">
                                <SelectValue placeholder="Выберите причину..." />
                            </SelectTrigger>
                            <SelectContent>
                                {REPORT_REASONS.map(r => (
                                    <SelectItem key={r.id} value={r.id}>{r.label}</SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>

                    <div>
                        <Label className="text-xs font-semibold text-muted-foreground mb-1.5 block">
                            Описание (необязательно)
                        </Label>
                        <Textarea
                            value={description}
                            onChange={e => setDescription(e.target.value)}
                            placeholder="Опишите подробнее, что именно нарушено..."
                            className="rounded-xl border-border/50 text-sm min-h-20 resize-none"
                            maxLength={500}
                        />
                        <p className="text-xs text-muted-foreground mt-1 text-right">{description.length}/500</p>
                    </div>

                    <div className="flex gap-2 pt-1">
                        <Button
                            variant="outline"
                            onClick={onClose}
                            className="flex-1 rounded-xl h-9 text-sm"
                        >
                            Отмена
                        </Button>
                        <Button
                            onClick={handleSubmit}
                            disabled={submitting || !reason}
                            className="flex-1 nexus-gradient border-0 text-white rounded-xl h-9 text-sm font-bold shadow-nexus"
                        >
                            Отправить
                        </Button>
                    </div>
                </div>
            </DialogContent>
        </Dialog>
    );
}
