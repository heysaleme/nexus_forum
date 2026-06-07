import { motion } from 'framer-motion';

export default function EmptyState({ icon: Icon, title, description, action }) {
    return (
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex flex-col items-center justify-center py-16 px-4 text-center"
        >
            {Icon && (
                <div className="w-16 h-16 nexus-gradient-soft rounded-3xl flex items-center justify-center mb-4">
                    <Icon className="w-8 h-8 text-primary" />
                </div>
            )}
            <h3 className="text-lg font-bold text-foreground mb-2">{title}</h3>
            {description && <p className="text-sm text-muted-foreground max-w-xs mb-4">{description}</p>}
            {action}
        </motion.div>
    );
}