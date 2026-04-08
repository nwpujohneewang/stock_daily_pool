import { useState, useCallback } from 'react';
import { Search, Inbox } from 'lucide-react';
import type { PoolData, ReclassifyResult } from '../../types';
import { CategorySection } from './CategorySection';

interface PoolContentProps {
  poolData: PoolData;
  activePool: 1 | 2;
  limitUpTopics: Set<string>;
  limitUpCategories: Set<string>;
  dataNotReady?: boolean;
  onManageTopic: (stock: ReclassifyResult) => void;
}

export function PoolContent({
  poolData,
  activePool,
  limitUpTopics,
  limitUpCategories,
  dataNotReady,
  onManageTopic,
}: PoolContentProps) {
  const [search, setSearch] = useState('');
  const [expandedTopics, setExpandedTopics] = useState<Record<string, boolean>>(
    {}
  );

  const toggleTopic = useCallback((catTopKey: string) => {
    setExpandedTopics((prev) => {
      const current = prev[catTopKey] === true;
      return { ...prev, [catTopKey]: !current };
    });
  }, []);

  const sortedCategories = Object.entries(poolData.categories).sort(
    ([a, aTopics], [b, bTopics]) => {
      if (a === '未分类') return 1;
      if (b === '未分类') return -1;
      
      // Calculate total stocks in each category (unique stocks)
      const aStocks = new Set(Object.values(aTopics).flat().map(s => s.ts_code)).size;
      const bStocks = new Set(Object.values(bTopics).flat().map(s => s.ts_code)).size;
      
      if (activePool === 2) {
        const aIn = limitUpCategories.has(a);
        const bIn = limitUpCategories.has(b);
        if (aIn !== bIn) return aIn ? -1 : 1;
        // If both are in or both are out, sort by stock count
        if (aStocks !== bStocks) return bStocks - aStocks;
      } else if (activePool === 1) {
        // For Pool 1, just sort by stock count
        if (aStocks !== bStocks) return bStocks - aStocks;
      }
      
      return a.localeCompare(b);
    }
  );

  return (
    <div className="bg-white border-none rounded-[20px] shadow-horizon-card overflow-hidden flex flex-col h-full">
      {/* Card Header */}
      <div className="px-5 py-3 sm:py-3.5 border-none flex flex-col sm:flex-row justify-between items-center gap-3 bg-white shrink-0">
        <div className="flex items-center gap-2">
          <span className="w-1.5 h-5 bg-horizon-brand rounded-full"></span>
          <h2 className="text-[16px] font-bold text-horizon-text-primary tracking-tight">
            {poolData.name} 明细
          </h2>
        </div>
        <div className="relative w-full sm:w-64">
          <Search
            className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
            size={14}
          />
          <input
            type="text"
            placeholder="搜索名称或代码..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="bg-slate-50 border border-slate-200 rounded-lg pl-9 pr-3 py-1.5 text-[13px] outline-none focus:ring-2 focus:ring-horizon-brand/20 focus:border-horizon-brand transition-all w-full"
          />
        </div>
      </div>

      {/* Categorized List */}
      <div className="px-3 pb-3 bg-slate-50/50 flex-1">
        {dataNotReady && (
          <div className="p-16 text-center text-slate-400 bg-white rounded-xl">
            <div className="flex justify-center mb-4 text-slate-300">
              <Inbox size={48} />
            </div>
            <div className="text-base font-medium text-slate-500 mb-2">当日数据未准备好</div>
            <div className="text-sm">请稍后刷新页面重试</div>
          </div>
        )}

        {!dataNotReady && sortedCategories.length === 0 && (
          <div className="p-16 text-center text-slate-400 bg-white rounded-xl">
            <div className="flex justify-center mb-4 text-slate-200">
              <Inbox size={48} />
            </div>
            <div className="text-sm">该日期暂无数据记录</div>
          </div>
        )}

        {!dataNotReady && sortedCategories.map(([category, topics]) => (
          <CategorySection
            key={category}
            category={category}
            topics={topics}
            expandedTopics={expandedTopics}
            activePool={activePool}
            search={search}
            limitUpTopics={limitUpTopics}
            onToggleTopic={toggleTopic}
            onManageTopic={onManageTopic}
          />
        ))}
      </div>
    </div>
  );
}
