import { memo } from 'react';
import type { ReclassifyResult } from '../../types';
import { TopicGroup } from './TopicGroup';

interface CategorySectionProps {
  category: string;
  topics: Record<string, ReclassifyResult[]>;
  expandedTopics: Record<string, boolean>;
  activePool: 1 | 2;
  search: string;
  limitUpTopics: Set<string>;
  onToggleTopic: (catTopKey: string) => void;
  onManageTopic: (stock: ReclassifyResult) => void;
}

export const CategorySection = memo(function CategorySection({
  category,
  topics,
  expandedTopics,
  activePool,
  search,
  limitUpTopics,
  onToggleTopic,
  onManageTopic,
}: CategorySectionProps) {
  const sortedTopics = Object.entries(topics).sort(
    ([aTopic, aStocks], [bTopic, bStocks]) => {
      // Priority 1: Put LimitUp Topics first (if in Pool 2)
      if (activePool === 2) {
        const aIn = limitUpTopics.has(aTopic);
        const bIn = limitUpTopics.has(bTopic);
        if (aIn !== bIn) return aIn ? -1 : 1;
      }

      // Priority 2: Sort by number of stocks in this topic (descending)
      if (aStocks.length !== bStocks.length) {
        return bStocks.length - aStocks.length;
      }

      return aTopic.localeCompare(bTopic);
    }
  );

  return (
    <div className="group/cat relative mb-1.5 mt-0.5">
      {/* 方案A：自定义背景色样式 */}
      <div
        className="px-3 py-1 font-semibold text-[13px] uppercase tracking-[0.16em] rounded-t-lg shadow-sm flex items-center gap-1.5"
        style={{ backgroundColor: '#dff4f6', color: '#4b7f86' }}
      >
        <div className="w-1 h-2.5 rounded-full" style={{ backgroundColor: '#8ccdd4' }}></div>
        {category}
      </div>

      {/* 方案B：白色背景 + 左侧粗边框 + 上下细边框样式 */}
      {/* <div className="px-6 py-3 bg-white text-slate-800 font-bold text-[14px] uppercase tracking-wider border-y border-r border-slate-200 border-l-4 border-l-blue-500 shadow-sm flex items-center rounded-t-xl">
        {category}
      </div> */}

      <div className="divide-y divide-slate-50 border-x border-b border-slate-200 rounded-b-xl bg-white overflow-hidden">
        {sortedTopics.map(([topicName, stocks]) => {
          const filteredStocks = stocks.filter(
            (s) =>
              !search ||
              s.name.includes(search) ||
              s.ts_code.includes(search)
          );

          const catTopKey = `${category}-${topicName}`;
          const isExpanded = expandedTopics[catTopKey] === true;

          return (
            <TopicGroup
              key={topicName}
              catTopKey={catTopKey}
              topicName={topicName}
              stocks={stocks}
              filteredStocks={filteredStocks}
              isExpanded={isExpanded}
              activePool={activePool}
              isLimitUpTopic={limitUpTopics.has(topicName)}
              onToggle={onToggleTopic}
              onManageTopic={onManageTopic}
            />
          );
        })}
      </div>
    </div>
  );
});
