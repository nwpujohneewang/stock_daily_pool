import clsx from 'clsx';
import { Flame } from 'lucide-react';

interface DaysBadgeProps {
  days: number;
  label?: string;
}

export function DaysBadge({ days, label }: DaysBadgeProps) {
  if (days <= 1) {
    return <span className="text-slate-400 text-[11px]">{label || `${days}天`}</span>;
  }

  const colorClass = days > 10
    ? 'bg-rose-200 text-rose-700'
    : days > 5
      ? 'bg-rose-100 text-rose-700'
      : 'bg-rose-50 text-rose-600';

  return (
    <span className={clsx(
      days > 3 ? 'text-[12px]' : 'text-[11px]',
      'inline-flex items-center gap-1 px-1.5 py-0 rounded font-semibold whitespace-nowrap',
      colorClass
    )}>
      {days > 5 ? (
        <Flame size={14} className="text-rose-600" />
      ) : days > 3 ? (
        <Flame size={11} className="text-rose-500" />
      ) : null}
      {label || `${days}天`}
    </span>
  );
}
