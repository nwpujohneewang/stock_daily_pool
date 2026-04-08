import { memo } from 'react';
import { ChevronDown } from 'lucide-react';
import clsx from 'clsx';
import type { ReclassifyResult } from '../../types';
import { StockRow } from './StockRow';

interface TopicGroupProps {
  catTopKey: string;
  topicName: string;
  stocks: ReclassifyResult[];
  filteredStocks: ReclassifyResult[];
  isExpanded: boolean;
  activePool: 1 | 2;
  isLimitUpTopic: boolean;
  onToggle: (catTopKey: string) => void;
  onManageTopic: (stock: ReclassifyResult) => void;
}

export const TopicGroup = memo(function TopicGroup({
  catTopKey,
  topicName,
  stocks,
  filteredStocks,
  isExpanded,
  activePool,
  isLimitUpTopic,
  onToggle,
  onManageTopic,
}: TopicGroupProps) {
  if (filteredStocks.length === 0) return null;

  return (
    <div className="bg-white">
      <div
        className="px-3 py-1 flex items-center justify-between cursor-pointer hover:bg-slate-50 transition-colors group/topic"
        onClick={() => onToggle(catTopKey)}
      >
        <div className="flex items-center gap-1.5">
          <span
            className={clsx(
              'text-slate-300 transition-transform duration-200',
              isExpanded ? 'rotate-0' : '-rotate-90'
            )}
          >
            <ChevronDown size={11} />
          </span>
          <span className="font-semibold text-horizon-text-primary text-[13px]">
            {topicName}
          </span>
          <span className="text-[10px] font-semibold px-1.5 py-0.5 rounded-full bg-slate-100 text-slate-500 border border-slate-200">
            {stocks.length}
          </span>
        </div>
      </div>

      {isExpanded && (
        <div className="px-3 pb-1.5">
          <div className="border border-slate-100 rounded-lg overflow-hidden shadow-sm">
            <table className="w-full text-left text-sm">
              <thead className="bg-slate-50/80 text-slate-400 font-semibold text-[10px] uppercase tracking-[0.14em] border-b border-slate-100">
                <tr>
                  <th className="px-2.5 py-1.5 w-40">股票</th>
                  <th className="px-2.5 py-1.5 text-center">涨跌幅</th>
                  <th className="px-2.5 py-1.5 text-center">价格</th>
                  {activePool === 1 && (
                    <th className="px-2.5 py-1.5 text-center">封板时间</th>
                  )}
                  {activePool === 1 && (
                    <th className="px-2.5 py-1.5 text-center">连板</th>
                  )}
                  {activePool === 1 && (
                    <th className="px-2.5 py-1.5 text-center">总市值</th>
                  )}
                  {activePool === 1 && (
                    <th className="px-2.5 py-1.5 text-center">成交量</th>
                  )}
                  {activePool === 1 && (
                    <th className="px-2.5 py-1.5 text-center">成交额</th>
                  )}
                  {activePool === 2 && (
                    <th className="px-2.5 py-1.5 text-center">连续&gt;5%</th>
                  )}
                  <th className="px-2.5 py-1.5 text-left">标签</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-50">
                {filteredStocks.map((stock) => (
                  <StockRow
                    key={stock.ts_code}
                    stock={stock}
                    activePool={activePool}
                    isHighlighted={activePool === 2 && isLimitUpTopic}
                    onManageTopic={onManageTopic}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
});
