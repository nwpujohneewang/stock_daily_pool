import { memo, useState } from 'react';
import clsx from 'clsx';
import type { ReclassifyResult } from '../../types';
import { DaysBadge } from './DaysBadge';

interface StockRowProps {
  stock: ReclassifyResult;
  activePool: 1 | 2;
  isHighlighted: boolean;
  onManageTopic: (stock: ReclassifyResult) => void;
}

function formatAmount(val?: number): string {
  if (!val) return '-';
  if (val >= 100000) return (val / 100000).toFixed(1) + '亿';
  if (val >= 100) return (val / 100).toFixed(1) + '万';
  return val.toFixed(0);
}

function formatMv(val?: number): string {
  if (!val) return '-';
  if (val >= 10000) return (val / 10000).toFixed(1) + '亿';
  return val.toFixed(0) + '万';
}

export const StockRow = memo(function StockRow({
  stock,
  activePool,
  isHighlighted,
  onManageTopic,
}: StockRowProps) {
  const [showAllTopics, setShowAllTopics] = useState(false);
  const topics = stock.all_topics || [];
  const displayTopics = showAllTopics ? topics : topics.slice(0, 3);
  const hasMore = topics.length > 3;

  return (
    <tr
      className={clsx(
        'hover:bg-blue-50/30 transition-colors',
        isHighlighted && 'bg-orange-50/40'
      )}
    >
      <td className="px-2.5 py-1 w-40">
        <div className="font-semibold text-[13px] text-horizon-text-primary whitespace-nowrap overflow-hidden text-ellipsis">{stock.name}</div>
        <div className="text-[10px] font-mono text-slate-400">
          {stock.ts_code}
        </div>
        <button
          onClick={() => onManageTopic(stock)}
          className="mt-1 text-[10px] font-bold text-emerald-600 hover:text-emerald-800"
        >
          修改 topic
        </button>
      </td>
      <td className="px-2.5 py-1 text-center">
        <span
          className={clsx(
            'font-semibold text-[13px]',
            stock.change_pct >= 9.9 ? 'text-rose-600' : 'text-orange-500'
          )}
        >
          {stock.change_pct > 0 ? '+' : ''}
          {stock.change_pct.toFixed(2)}%
        </span>
      </td>
      <td className="px-2.5 py-1 text-center text-[13px] font-medium text-slate-600">
        {stock.price.toFixed(2)}
      </td>
      {activePool === 1 && (
        <td className="px-2.5 py-1 text-center">
          {stock.first_time ? (
            <div className="text-[11px] leading-tight">
              <div className="font-medium text-slate-700">{stock.first_time}</div>
              {stock.last_time && stock.last_time !== stock.first_time && (
                <div className="text-slate-400">→ {stock.last_time}</div>
              )}
            </div>
          ) : (
            <span className="text-slate-300">-</span>
          )}
        </td>
      )}
      {activePool === 1 && (
        <td className="px-2.5 py-1 text-center">
          {stock.limit_times && stock.limit_times > 0 ? (
            <DaysBadge days={stock.limit_times} label={stock.limit_times_display || `${stock.limit_times}连板`} />
          ) : (
            <span className="text-slate-300">-</span>
          )}
        </td>
      )}
      {activePool === 1 && (
        <td className="px-2.5 py-1 text-center text-[11px] text-slate-600">
          {formatMv(stock.total_mv)}
        </td>
      )}
      {activePool === 1 && (
        <td className="px-2.5 py-1 text-center text-[11px] text-slate-600">
          {formatAmount(stock.vol)}
        </td>
      )}
      {activePool === 1 && (
        <td className="px-2.5 py-1 text-center text-[11px] text-slate-600">
          {formatAmount(stock.amount)}
        </td>
      )}
      {activePool === 2 && (
        <td className="px-2.5 py-1 text-center">
          <DaysBadge days={stock.consecutive_strong_days || 1} />
        </td>
      )}
      <td className="px-2.5 py-1 w-60 max-w-[15rem]">
        <div className="flex flex-wrap gap-1 justify-start items-center">
          {displayTopics.map((topic, i) => (
            <span
              key={i}
              className="px-1.5 py-0.5 bg-blue-50/70 text-blue-600 rounded text-[10px] font-semibold border border-blue-100 whitespace-nowrap"
              title={`分类: ${topic.category} | 命中: ${topic.hit_count}`}
            >
              {topic.topic_name}
            </span>
          ))}
          {hasMore && (
            <button
              onClick={() => setShowAllTopics(!showAllTopics)}
              className="text-[10px] font-semibold text-slate-500 hover:text-blue-600 hover:bg-blue-50 px-1 py-0.5 rounded transition-colors"
            >
              {showAllTopics ? '收起' : `+${topics.length - 3}`}
            </button>
          )}
        </div>
      </td>
    </tr>
  );
});
