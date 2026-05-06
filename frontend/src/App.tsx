import { useState, useEffect, useMemo } from 'react';
import axios from 'axios';
import { RefreshCw, BarChart3, Inbox, BookOpen, Plus, Trash2, Edit2, X, Check, Download, Search, Settings, Eye, EyeOff } from 'lucide-react';
import clsx from 'clsx';
import { PoolContent, DaysBadge } from './components/monitor';
import { TopicRelationManagerModal } from './components/stock/TopicRelationManagerModal';
import type { ReclassifyResult, TopicRelation } from './types';

// Types
interface TopicDictionary {
  id: number;
  raw_topic_name: string;
  normalized_name: string;
  category: string;
}

interface StockBasicInfo {
  id: number;
  ts_code: string;
  symbol: string;
  name: string;
  exchange: string;
  board_code: string;
  industry?: string;
  is_st: boolean;
  list_date?: string;
}

interface StockDetail {
  info: StockBasicInfo;
  topics: TopicRelation[];
}

interface TopicOption {
  id: number;
  name: string;
  category: string;
}

interface RebuildTopicInfo {
  name: string;
  normalized_name: string;
  stocks_count: number;
}

interface RebuildResult {
  start_date: string;
  end_date: string;
  topics_count: number;
  relations_count: number;
  topics: RebuildTopicInfo[];
}

interface ApiErrorPayload {
  message?: string;
  msg?: string;
  data?: unknown;
}

interface AutocompleteInputProps {
  value: string;
  onChange: (val: string) => void;
  onEnter: () => void;
  placeholder: string;
  icon: React.ReactNode;
  suggestions: string[];
}

function AutocompleteInput({ value, onChange, onEnter, placeholder, icon, suggestions }: AutocompleteInputProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isFocused, setIsFocused] = useState(false);

  // Prefix match filter
  const filteredSuggestions = suggestions.filter(s =>
    s.toLowerCase().startsWith(value.toLowerCase())
  );

  return (
    <div className="relative w-full">
      <div className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
        {icon}
      </div>
      <input
        type="text"
        placeholder={placeholder}
        className="w-full bg-slate-50 border border-slate-200 rounded-xl pl-10 pr-4 py-2 text-sm outline-none focus:ring-2 focus:ring-horizon-brand/20 focus:border-horizon-brand transition-all"
        value={value}
        onChange={e => {
          onChange(e.target.value);
          setIsOpen(true);
        }}
        onFocus={() => {
          setIsFocused(true);
          setIsOpen(true);
        }}
        onBlur={() => {
          setIsFocused(false);
          // Small delay to allow click event on suggestion to fire before closing
          setTimeout(() => setIsOpen(false), 200);
        }}
        onKeyDown={e => {
          if (e.key === 'Enter') {
            setIsOpen(false);
            onEnter();
          }
        }}
      />

      {isOpen && isFocused && (
        <div className="absolute top-full left-0 w-full mt-2 bg-white border border-slate-100 rounded-[16px] shadow-horizon-card z-[60] max-h-60 overflow-y-auto">
          {filteredSuggestions.length > 0 ? (
            <ul className="py-2">
              {filteredSuggestions.map((suggestion, idx) => (
                <li
                  key={idx}
                  className="px-4 py-2.5 text-sm cursor-pointer hover:bg-horizon-bg hover:text-horizon-brand text-horizon-text-primary transition-colors"
                  onMouseDown={(e) => {
                    // Prevent input blur before click registers
                    e.preventDefault();
                    onChange(suggestion);
                    setIsOpen(false);
                  }}
                >
                  {suggestion}
                </li>
              ))}
            </ul>
          ) : (
            <div className="px-4 py-4 text-sm text-slate-400 text-center italic">无匹配项</div>
          )}
        </div>
      )}
    </div>
  );
}

interface AddDictRowProps {
  onSave: (payload: Partial<TopicDictionary>) => void;
  onCancel: () => void;
}

function AddDictRow({ onSave, onCancel }: AddDictRowProps) {
  const [rawTopicName, setRawTopicName] = useState('');
  const [normalizedName, setNormalizedName] = useState('');
  const [category, setCategory] = useState('');

  return (
    <tr className="bg-blue-50/50">
      <td className="px-6 py-4">
        <input
          autoFocus
          placeholder="例如: 华为海思"
          className="w-full bg-white border border-blue-200 rounded px-3 py-1.5 outline-none focus:border-blue-500"
          value={rawTopicName}
          onChange={e => setRawTopicName(e.target.value)}
        />
      </td>
      <td className="px-6 py-4">
        <input
          placeholder="例如: 半导体"
          className="w-full bg-white border border-blue-200 rounded px-3 py-1.5 outline-none focus:border-blue-500"
          value={normalizedName}
          onChange={e => setNormalizedName(e.target.value)}
        />
      </td>
      <td className="px-6 py-4">
        <input
          placeholder="例如: 电子"
          className="w-full bg-white border border-blue-200 rounded px-3 py-1.5 outline-none focus:border-blue-500"
          value={category}
          onChange={e => setCategory(e.target.value)}
        />
      </td>
      <td className="px-6 py-4 text-right space-x-2">
        <button
          onClick={() => onSave({
            raw_topic_name: rawTopicName,
            normalized_name: normalizedName,
            category,
          })}
          className="text-blue-600 hover:text-blue-800 p-1.5 rounded-lg hover:bg-blue-100 transition-colors"
        >
          <Check size={18} />
        </button>
        <button
          onClick={onCancel}
          className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg hover:bg-slate-200 transition-colors"
        >
          <X size={18} />
        </button>
      </td>
    </tr>
  );
}

function getApiErrorMessage(err: unknown, fallback: string) {
  if (axios.isAxiosError<ApiErrorPayload>(err)) {
    return err.response?.data?.message || err.response?.data?.msg || fallback;
  }
  return fallback;
}

export default function App() {
  const [date, setDate] = useState(() => {
    const today = new Date();
    return today.toISOString().split('T')[0];
  });
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<ReclassifyResult[]>([]);
  const [yesterdayStrongData, setYesterdayStrongData] = useState<ReclassifyResult[]>([]);
  const [isTrading, setIsTrading] = useState(false);
  const [dataNotReady, setDataNotReady] = useState(false);
  const [activePool, setActivePool] = useState<1 | 2 | 3>(1);

  // Navigation
  const [activeTab, setActiveTab] = useState<'monitor' | 'dictionary' | 'stock' | 'tools'>('monitor');

  // Stock Search State
  const [stockQuery, setStockSearchQuery] = useState('');
  const [stockTopicFilter, setStockTopicFilter] = useState('');
  const [stockCategoryFilter, setStockCategoryFilter] = useState('');
  const [stockPage, setStockPage] = useState(1);
  const [stockPageSize] = useState(20);
  const [stockTotal, setStockTotal] = useState(0);
  const [stockResults, setStockSearchResults] = useState<StockBasicInfo[]>([]);
  const [selectedStock, setSelectedStock] = useState<StockDetail | null>(null);
  const [topicManagerStock, setTopicManagerStock] = useState<{ ts_code: string; name: string } | null>(null);
  const [topicManagerRelations, setTopicManagerRelations] = useState<TopicRelation[]>([]);
  const [allTopicsForPicker, setAllTopicsForPicker] = useState<TopicOption[]>([]);
  const [isSearchingStock, setIsSearchingStock] = useState(false);
  const [syncingMarket, setSyncingMarket] = useState(false);
  const [syncMarketError, setSyncMarketError] = useState<string | null>(null);

  // Rebuild State
  const [rebuildStartDate, setRebuildStartDate] = useState('');
  const [rebuildEndDate, setRebuildEndDate] = useState('');
  const [rebuildLoading, setRebuildLoading] = useState(false);
  const [rebuildResult, setRebuildResult] = useState<RebuildResult | null>(null);
  const [rebuildError, setRebuildError] = useState<string | null>(null);
  const [showRebuildModal, setShowRebuildModal] = useState(false);

  // Crawl Error State (used by MissingTopics modal)
  const [crawlMissingTopics, setCrawlMissingTopics] = useState<string[]>([]);
  const [crawlFailedDate, setCrawlFailedDate] = useState<string>('');
  const [showMissingTopicsModal, setShowMissingTopicsModal] = useState(false);

  // Crawl Jiuyan modal state
  const [showCrawlModal, setShowCrawlModal] = useState(false);
  const [curlInput, setCurlInput] = useState('');
  const [crawlLoading, setCrawlLoading] = useState(false);
  const [crawlResult, setCrawlResult] = useState<{ date: string; topics_count: number; stocks_count: number } | null>(null);
  const [crawlError, setCrawlError] = useState<string | null>(null);
  // Note: setCrawlMissingTopics and setCrawlFailedDate will be used when integrating crawl error handling
  void setCrawlMissingTopics; void setCrawlFailedDate;
  // Tools: Snapshot trigger
  const [snapshotStartDate, setSnapshotStartDate] = useState('');
  const [snapshotEndDate, setSnapshotEndDate] = useState('');
  const [snapshotLoading, setSnapshotLoading] = useState(false);
  const [snapshotError, setSnapshotError] = useState<string | null>(null);
  const [snapshotSuccessMsg, setSnapshotSuccessMsg] = useState<string | null>(null);

  // LLM Classify History Test
  const [llmClassifyDate, setLlmClassifyDate] = useState(() => {
    const d = new Date();
    d.setDate(d.getDate() - 1);
    return d.toISOString().split('T')[0];
  });
  const [llmClassifyLoading, setLlmClassifyLoading] = useState(false);
  const [llmClassifyResult, setLlmClassifyResult] = useState<{ status: string; date: string; elapsed_ms: number; total: number; results: { ts_code: string; name: string; topic_name: string; topic_id: number; category: string; confidence: number }[] } | null>(null);
  const [llmClassifyError, setLlmClassifyError] = useState<string | null>(null);

  // Hot Spot Fetch News
  const [fetchNewsLoading, setFetchNewsLoading] = useState(false);
  const [fetchNewsResult, setFetchNewsResult] = useState<{ total: number; news: { title: string; content: string; source: string; importance: string; published_at: string }[] } | null>(null);
  const [fetchNewsError, setFetchNewsError] = useState<string | null>(null);
  const [fetchNewsSource, setFetchNewsSource] = useState<'all' | 'cls' | 'eastmoney'>('all');
  const [fetchNewsImportance, setFetchNewsImportance] = useState<'high' | 'medium' | 'low'>('medium');

  const fetchStocks = async (page = 1) => {
    setIsSearchingStock(true);
    setStockPage(page);
    try {
      const res = await axios.get(`/api/v1/stocks/search`, {
        params: {
          q: stockQuery,
          topic: stockTopicFilter,
          category: stockCategoryFilter,
          page: page,
          pageSize: stockPageSize
        },
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        setStockSearchResults(res.data.data.items || []);
        setStockTotal(res.data.data.total || 0);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setIsSearchingStock(false);
    }
  };

  const fetchStockDetail = async (tsCode: string) => {
    try {
      const res = await axios.get(`/api/v1/stocks/${tsCode}`, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        setSelectedStock(res.data.data);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const fetchTopicsForPicker = async () => {
    try {
      const res = await axios.get('/api/v1/topics', {
        params: { page: 1, page_size: 2000 },
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        setAllTopicsForPicker((res.data.data.items || []).map((topic: TopicOption) => ({
          id: topic.id,
          name: topic.name,
          category: topic.category,
        })));
      }
    } catch (err) {
      console.error(err);
    }
  };

  const openTopicManager = async (stock: { ts_code: string; name: string }, relations?: TopicRelation[]) => {
    if (allTopicsForPicker.length === 0) {
      await fetchTopicsForPicker();
    }
    setTopicManagerStock(stock);
    if (relations) {
      setTopicManagerRelations(relations);
    } else if (selectedStock?.info.ts_code === stock.ts_code) {
      setTopicManagerRelations(selectedStock.topics || []);
    } else {
      setTopicManagerRelations([]);
    }
    if (!relations) {
      await fetchStockDetail(stock.ts_code);
    }
  };

  const handleTopicRelationsUpdated = async (relations: TopicRelation[]) => {
    setTopicManagerRelations(relations);
    if (topicManagerStock) {
      if (selectedStock?.info.ts_code === topicManagerStock.ts_code) {
        setSelectedStock({
          ...selectedStock,
          topics: relations,
        });
      }
      if (activeTab === 'stock') {
        await fetchStockDetail(topicManagerStock.ts_code);
      }
      if (activeTab === 'monitor') {
        await fetchData(false);
      }
    }
  };

  const handleToolsSyncMarket = async (date?: string) => {
    setSyncMarketError(null);
    setSyncingMarket(true);
    try {
      const res = await axios.post('/api/v1/stocks/sync_basic', date ? { date } : {}, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        setSyncMarketError(null);
      } else {
        setSyncMarketError(res.data?.message || res.data?.msg || '同步市场失败');
      }
    } catch (err: unknown) {
      setSyncMarketError(getApiErrorMessage(err, '同步市场失败'));
    } finally {
      setSyncingMarket(false);
    }
  };
  const handleTriggerSnapshot = async () => {
    setSnapshotError(null);
    setSnapshotSuccessMsg(null);
    setSnapshotLoading(true);
    try {
      if (!snapshotStartDate || !snapshotEndDate) {
        setSnapshotError('请选择开始日期和结束日期');
        setSnapshotLoading(false);
        return;
      }
      const res = await axios.post('/api/v1/snapshot/trigger-range', {
        start_date: snapshotStartDate,
        end_date: snapshotEndDate,
      }, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        const data = res.data.data || {};
        const processed = Array.isArray(data.processed_dates) ? data.processed_dates.length : 0;
        const skipped = Array.isArray(data.skipped_dates) ? data.skipped_dates.length : 0;
        const failed = Array.isArray(data.failed_dates) ? data.failed_dates.length : 0;
        setSnapshotSuccessMsg(`触发完成：成功 ${processed} 天，跳过 ${skipped} 天，失败 ${failed} 天`);
      } else {
        setSnapshotError(res.data?.message || res.data?.msg || '触发失败');
      }
    } catch (err: unknown) {
      setSnapshotError(getApiErrorMessage(err, '触发失败'));
    } finally {
      setSnapshotLoading(false);
    }
  };

  const handleLLMClassifyHistory = async () => {
    if (!llmClassifyDate) {
      setLlmClassifyError('请选择日期');
      return;
    }
    setLlmClassifyLoading(true);
    setLlmClassifyError(null);
    setLlmClassifyResult(null);
    try {
      const res = await axios.post(`/api/v1/llm-classify/run-history?date=${llmClassifyDate}`, {}, {
        headers: { 'X-API-Key': 'test-api-key' },
        timeout: 600000,
      });
      if (res.data && res.data.code === 0) {
        setLlmClassifyResult(res.data.data);
      } else {
        setLlmClassifyError(res.data?.message || res.data?.msg || 'LLM分类失败');
      }
    } catch (err: unknown) {
      setLlmClassifyError(getApiErrorMessage(err, 'LLM分类请求失败'));
    } finally {
      setLlmClassifyLoading(false);
    }
  };

  const handleFetchNews = async () => {
    setFetchNewsLoading(true);
    setFetchNewsError(null);
    setFetchNewsResult(null);
    try {
      const res = await axios.get('/api/v1/hot-spot/fetch-news', {
        params: { source: fetchNewsSource, importance: fetchNewsImportance },
        headers: { 'X-API-Key': 'test-api-key' },
        timeout: 30000,
      });
      if (res.data && res.data.code === 0) {
        setFetchNewsResult(res.data.data);
      } else {
        setFetchNewsError(res.data?.message || res.data?.msg || '获取新闻失败');
      }
    } catch (err: unknown) {
      setFetchNewsError(getApiErrorMessage(err, '获取新闻失败'));
    } finally {
      setFetchNewsLoading(false);
    }
  };

  const handleRebuild = async () => {
    if (!rebuildStartDate || !rebuildEndDate) {
      setRebuildError('请选择日期范围');
      return;
    }
    setRebuildLoading(true);
    setRebuildError(null);
    setRebuildResult(null);
    try {
      const res = await axios.post('/api/v1/crawl/rebuild', {
        start_date: rebuildStartDate,
        end_date: rebuildEndDate
      }, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        setRebuildResult(res.data.data);
      } else {
        setRebuildError(res.data.message || res.data.msg || '重建失败');
      }
    } catch (err: unknown) {
      if (axios.isAxiosError<ApiErrorPayload>(err)) {
        setRebuildError(getApiErrorMessage(err, '重建失败'));
        if (Array.isArray(err.response?.data?.data)) {
          setRebuildError(`缺失 topic 映射: ${err.response?.data?.data.join(', ')}`);
        }
      } else {
        setRebuildError('重建失败');
      }
    } finally {
      setRebuildLoading(false);
    }
  };

  // Handle Crawl Jiuyan
  const handleCrawlJiuyan = async () => {
    if (!curlInput.trim()) {
      setCrawlError('请输入 curl 命令');
      return;
    }
    setCrawlLoading(true);
    setCrawlError(null);
    setCrawlResult(null);
    try {
      const res = await axios.post('/api/v1/crawl/jiuyan', {
        curl: curlInput
      }, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        setCrawlResult(res.data.data);
      } else {
        setCrawlError(res.data.message || res.data.msg || '爬取失败');
      }
    } catch (err: unknown) {
      if (axios.isAxiosError<ApiErrorPayload>(err)) {
        setCrawlError(getApiErrorMessage(err, '爬取失败'));
        if (Array.isArray(err.response?.data?.data)) {
          setCrawlMissingTopics(err.response.data.data);
          setShowCrawlModal(false);
          setShowMissingTopicsModal(true);
        }
      } else {
        setCrawlError('爬取失败');
      }
    } finally {
      setCrawlLoading(false);
    }
  };

  useEffect(() => {
    if (activeTab === 'stock') {
      fetchStocks(1);
    }
  }, [activeTab, stockTopicFilter, stockCategoryFilter]);

  // Dictionary State
  const [dictionary, setDictionary] = useState<TopicDictionary[]>([]);
  const [dictSearch, setDictSearch] = useState('');
  const [isAddingDict, setIsAddingDict] = useState(false);
  const [editingDictId, setEditingDictId] = useState<number | null>(null);
  const [hiddenDictIds, setHiddenDictIds] = useState<number[]>([]);
  const [newDict, setNewDict] = useState<Partial<TopicDictionary>>({
    raw_topic_name: '',
    normalized_name: '',
    category: '',
  });

  const fetchDictionary = async () => {
    try {
      const res = await axios.get('/api/v1/topic-dictionary', {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        setDictionary(res.data.data || []);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleAddDict = async (payload: Partial<TopicDictionary>) => {
    if (!payload.raw_topic_name || !payload.normalized_name) return;
    try {
      const res = await axios.post('/api/v1/topic-dictionary', payload, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        fetchDictionary();
        setIsAddingDict(false);
        setNewDict({ raw_topic_name: '', normalized_name: '', category: '' });
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleUpdateDict = async (id: number) => {
    try {
      const res = await axios.post(`/api/v1/topic-dictionary/update/${id}`, newDict, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        fetchDictionary();
        setEditingDictId(null);
        setNewDict({ raw_topic_name: '', normalized_name: '', category: '' });
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleDeleteDict = async (id: number) => {
    if (!confirm('确定删除该词条吗？')) return;
    try {
      const res = await axios.post(`/api/v1/topic-dictionary/delete/${id}`, {}, {
        headers: { 'X-API-Key': 'test-api-key' }
      });
      if (res.data && res.data.code === 0) {
        fetchDictionary();
      }
    } catch (err) {
      console.error(err);
    }
  };

  const fetchData = async (showLoading = true) => {
    if (showLoading) setLoading(true);
    try {
      const res = await axios.get(`/api/v1/pool/reclassify?date=${date}`, {
        headers: {
          'X-API-Key': 'test-api-key'
        }
      });
      if (res.data && res.data.code === 0) {
        setData(res.data.data.items || []);
        setYesterdayStrongData(res.data.data.yesterday_strong_items || []);
        setIsTrading(res.data.data.is_trading || false);
        setDataNotReady(false);
      } else if (res.data && res.data.code === 4001) {
        // Realtime data not ready
        setData([]);
        setYesterdayStrongData([]);
        setIsTrading(false);
        setDataNotReady(true);
      }
    } catch (err) {
      console.error(err);
    } finally {
      if (showLoading) setLoading(false);
    }
  };

  useEffect(() => {
    fetchDictionary();
  }, []);

  const uniqueTopics = useMemo(() => {
    const topics = new Set<string>();
    dictionary.forEach(d => {
      if (d.normalized_name) topics.add(d.normalized_name);
    });
    return Array.from(topics).sort();
  }, [dictionary]);

  const uniqueCategories = useMemo(() => {
    const categories = new Set<string>();
    dictionary.forEach(d => {
      if (d.category) categories.add(d.category);
    });
    return Array.from(categories).sort();
  }, [dictionary]);

  const filteredDictionary = useMemo(() => {
    const visibleDictionary = dictionary.filter(d => !hiddenDictIds.includes(d.id));
    if (!dictSearch) return visibleDictionary;
    return visibleDictionary.filter(d =>
      d.raw_topic_name.includes(dictSearch) ||
      d.normalized_name.includes(dictSearch) ||
      d.category.includes(dictSearch)
    );
  }, [dictionary, dictSearch, hiddenDictIds]);

  // URL 参数预填充 rebuild 日期
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const startDate = params.get('rebuild_start_date');
    const endDate = params.get('rebuild_end_date');
    if (startDate && endDate) {
      setRebuildStartDate(startDate);
      setRebuildEndDate(endDate);
      setActiveTab('dictionary');
      setShowRebuildModal(true);
    }
  }, []);

  useEffect(() => {
    // Initial fetch when date changes
    fetchData(true);
  }, [date]);

  useEffect(() => {
    // Setup polling if today AND it is trading time AND data is ready
    const today = new Date().toISOString().split('T')[0];
    let interval: ReturnType<typeof setInterval> | undefined;

    if (date === today && isTrading && !dataNotReady) {
      interval = setInterval(() => {
        fetchData(false); // Background update
      }, 10000);
    }

    return () => {
      if (interval) clearInterval(interval);
    };
  }, [date, isTrading, dataNotReady]);

  const processedData = useMemo(() => {
    const pools = {
      1: { name: '涨停池', categories: {} as Record<string, Record<string, ReclassifyResult[]>>, total: 0 },
      2: { name: '5%池', categories: {} as Record<string, Record<string, ReclassifyResult[]>>, total: 0 },
    };
    const limitUpTopics = new Set<string>();
    const limitUpCategories = new Set<string>();

    data.forEach(stock => {
      if (stock.pool_type !== 1 && stock.pool_type !== 2) return;

      const pool = pools[stock.pool_type];
      pool.total++;

      const topics = stock.topics && stock.topics.length > 0
        ? stock.topics
        : [{ topic_name: '未分类', category: '未分类' } as TopicRelation];

      topics.forEach(topic => {
        const cat = topic.category || '未分类';
        const top = topic.topic_name || '未分类';

        if (stock.pool_type === 1) {
          limitUpTopics.add(top);
          limitUpCategories.add(cat);
        }

        if (!pool.categories[cat]) {
          pool.categories[cat] = {};
        }
        if (!pool.categories[cat][top]) {
          pool.categories[cat][top] = [];
        }

        if (!pool.categories[cat][top].find(s => s.ts_code === stock.ts_code)) {
          pool.categories[cat][top].push(stock);
        }
      });
    });

    // Sort the arrays within yesterdayStrongData directly
    const sortedYesterdayStrong = [...yesterdayStrongData].sort((a, b) => {
      // For yesterday strong, if there are limit up categories, put stocks with those categories first
      const aCategories = new Set((a.topics || []).map(t => t.category || '未分类'));
      const bCategories = new Set((b.topics || []).map(t => t.category || '未分类'));

      let aHasLimitUpCat = false;
      let bHasLimitUpCat = false;

      aCategories.forEach(cat => { if (limitUpCategories.has(cat)) aHasLimitUpCat = true; });
      bCategories.forEach(cat => { if (limitUpCategories.has(cat)) bHasLimitUpCat = true; });

      if (aHasLimitUpCat !== bHasLimitUpCat) {
        return aHasLimitUpCat ? -1 : 1;
      }

      // If neither or both have limit up categories, maintain original sort or sort by some other metric
      return (b.change_pct || 0) - (a.change_pct || 0);
    });

    return { pools, limitUpTopics, limitUpCategories, sortedYesterdayStrong };
  }, [data, yesterdayStrongData]);

  const yesterdayStrongLimitUps = useMemo(() => {
    return processedData.sortedYesterdayStrong.filter(s => (s.limit_times || 0) > 0 || s.is_limit_up);
  }, [processedData.sortedYesterdayStrong]);
  const yesterdayStrongNonLimit = useMemo(() => {
    return processedData.sortedYesterdayStrong.filter(s => !((s.limit_times || 0) > 0 || s.is_limit_up));
  }, [processedData.sortedYesterdayStrong]);
  const [collapseYesterdayLimitUps, setCollapseYesterdayLimitUps] = useState(false);
  const [collapseYesterdayNonLimit, setCollapseYesterdayNonLimit] = useState(false);

  const currentPoolData = activePool !== 3 ? processedData.pools[activePool] : null;

  return (
    <div className="min-h-screen bg-horizon-bg text-horizon-text-primary font-sans selection:bg-blue-100">
      {/* Sidebar / Navigation */}
      <nav className="fixed left-0 top-0 h-full w-20 bg-white flex flex-col items-center py-8 gap-8 rounded-r-[20px] shadow-horizon-card z-50">
        <div className="w-12 h-12 bg-horizon-brand rounded-2xl flex items-center justify-center text-white shadow-lg shadow-horizon-brand/20">
          <BarChart3 size={24} />
        </div>
        <div className="flex flex-col gap-6">
          <button
            onClick={() => setActiveTab('monitor')}
            className={clsx(
              "p-3 rounded-xl transition-all group relative",
              activeTab === 'monitor' ? "bg-horizon-bg text-horizon-brand" : "text-horizon-text-secondary hover:text-horizon-brand"
            )}
            title="实时监控"
          >
            <BarChart3 size={24} />
            {activeTab === 'monitor' && <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-horizon-brand rounded-r-full" />}
          </button>
          <button
            onClick={() => setActiveTab('dictionary')}
            className={clsx(
              "p-3 rounded-xl transition-all group relative",
              activeTab === 'dictionary' ? "bg-horizon-bg text-horizon-brand" : "text-horizon-text-secondary hover:text-horizon-brand"
            )}
            title="主题词典"
          >
            <BookOpen size={24} />
            {activeTab === 'dictionary' && <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-horizon-brand rounded-r-full" />}
          </button>
          <button
            onClick={() => setActiveTab('stock')}
            className={clsx(
              "p-3 rounded-xl transition-all group relative",
              activeTab === 'stock' ? "bg-horizon-bg text-horizon-brand" : "text-horizon-text-secondary hover:text-horizon-brand"
            )}
            title="股票查询"
          >
            <Search size={24} />
            {activeTab === 'stock' && <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-horizon-brand rounded-r-full" />}
          </button>
          <button
            onClick={() => setActiveTab('tools')}
            className={clsx(
              "p-3 rounded-xl transition-all group relative",
              activeTab === 'tools' ? "bg-horizon-bg text-horizon-brand" : "text-horizon-text-secondary hover:text-horizon-brand"
            )}
            title="工具"
          >
            <Settings size={24} />
            {activeTab === 'tools' && <div className="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-horizon-brand rounded-r-full" />}
          </button>
        </div>
      </nav>

      <div className="pl-20 min-h-screen">
        <div className="max-w-7xl mx-auto p-6 space-y-6">
          {activeTab === 'monitor' && (
            <>
              {/* Monitor Header */}
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div>
                  <h1 className="text-[24px] font-bold text-horizon-text-primary tracking-tight flex items-center gap-2">
                    股票分类结果
                  </h1>
                </div>
                <div className="flex items-center gap-4">
                  <div className="flex items-center gap-2 px-3 py-1.5 bg-white border border-slate-200 rounded-lg shadow-sm">
                    <span className={clsx(
                      "w-2 h-2 rounded-full",
                      isTrading ? "bg-emerald-500 animate-pulse shadow-[0_0_8px_rgba(16,185,129,0.6)]" : "bg-slate-300"
                    )}></span>
                    <span className={clsx(
                      "text-xs font-bold tracking-wide",
                      isTrading ? "text-emerald-600" : "text-slate-500"
                    )}>
                      {isTrading ? "交易进行中" : "休市中"}
                    </span>
                  </div>
                  <div className="h-8 w-px bg-slate-200 mx-1"></div>
                  <div className="flex items-center gap-3">
                    <input
                      type="date"
                      value={date}
                      onChange={e => setDate(e.target.value)}
                      className="bg-white border border-slate-200 rounded-lg px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all shadow-sm"
                    />
                    <button
                      onClick={() => fetchData(true)}
                      disabled={loading}
                      className="flex items-center gap-2 bg-horizon-brand hover:bg-horizon-brand-hover text-white px-4 py-2 rounded-[12px] text-sm font-semibold transition-all shadow-md disabled:opacity-50"
                    >
                      <RefreshCw size={14} className={clsx(loading && "animate-spin")} />
                      刷新数据
                    </button>
                  </div>
                </div>
              </div>

              {/* Summary Stats */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="bg-white border-none rounded-[20px] p-5 shadow-horizon-card">
                  <div className="text-[12px] font-bold text-horizon-text-secondary uppercase tracking-wider mb-1">涨停池</div>
                  <div className="text-3xl font-bold text-rose-500">{processedData.pools[1].total}</div>
                </div>
                <div className="bg-white border-none rounded-[20px] p-5 shadow-horizon-card">
                  <div className="text-[12px] font-bold text-horizon-text-secondary uppercase tracking-wider mb-1">5%池</div>
                  <div className="text-3xl font-bold text-orange-500">{processedData.pools[2].total}</div>
                </div>
                <div className="bg-white border-none rounded-[20px] p-5 shadow-horizon-card">
                  <div className="text-[12px] font-bold text-horizon-text-secondary uppercase tracking-wider mb-1">昨日强势</div>
                  <div className="text-3xl font-bold text-amber-500">{yesterdayStrongData.length}</div>
                </div>
              </div>

              {/* Tab Selection */}
              <div className="flex p-1 bg-white rounded-[16px] w-fit shadow-sm">
                {[1, 2, 3].map((type) => (
                  <button
                    key={type}
                    onClick={() => setActivePool(type as 1 | 2 | 3)}
                    className={clsx(
                      "px-6 py-2 rounded-lg text-sm font-bold transition-all flex items-center gap-2",
                      activePool === type
                        ? "bg-horizon-bg text-horizon-brand shadow-sm"
                        : "text-horizon-text-secondary hover:text-horizon-brand"
                    )}
                  >
                    {type === 1 ? '涨停池' : type === 2 ? '5%池' : '昨日强势'}
                    <span className={clsx(
                      "text-[10px] px-1.5 py-0.5 rounded-md font-bold",
                      activePool === type ? "bg-horizon-brand text-white" : "bg-horizon-bg text-horizon-text-secondary"
                    )}>
                      {type === 3 ? yesterdayStrongData.length : processedData.pools[type as 1 | 2].total}
                    </span>
                  </button>
                ))}
              </div>

              {/* Main Content Card for Pool 1 and 2 */}
              {(activePool === 1 || activePool === 2) && currentPoolData && (
                <div className="flex-1 overflow-hidden">
                  <PoolContent
                    poolData={currentPoolData}
                    activePool={activePool}
                    limitUpTopics={processedData.limitUpTopics}
                    limitUpCategories={processedData.limitUpCategories}
                    dataNotReady={dataNotReady}
                    onManageTopic={(stock) => openTopicManager({ ts_code: stock.ts_code, name: stock.name }, stock.all_topics || stock.topics || [])}
                  />
                </div>
              )}

              {/* Yesterday Strong Pool (Pool 3) */}
              {activePool === 3 && (
                <div className="bg-white border-none rounded-[20px] shadow-horizon-card overflow-hidden">
                  <div className="p-6">
                    <div className="flex items-center gap-2 mb-6">
                      <span className="w-2 h-6 bg-amber-500 rounded-full"></span>
                      <h2 className="text-[20px] font-bold text-horizon-text-primary tracking-tight">昨日强势 明细</h2>
                    </div>

                    {yesterdayStrongData.length === 0 ? (
                      <div className="p-16 text-center text-slate-400">
                        <div className="flex justify-center mb-4 text-slate-200">
                          <Inbox size={48} />
                        </div>
                        <div className="text-sm">暂无昨日强势数据</div>
                      </div>
                    ) : (
                      <div className="space-y-6">
                        {/* 昨日涨停 */}
                        <div>
                          <button
                            onClick={() => setCollapseYesterdayLimitUps(v => !v)}
                            className="w-full flex items-center justify-between mb-3 text-left"
                          >
                            <h3 className="text-sm font-bold text-rose-600">昨日涨停（{yesterdayStrongLimitUps.length}）</h3>
                            <span className="text-xs text-slate-500">{collapseYesterdayLimitUps ? '展开' : '收起'}</span>
                          </button>
                          {yesterdayStrongLimitUps.length === 0 ? (
                            <div className="p-6 text-center text-slate-400 border border-slate-100 rounded-xl">无</div>
                          ) : collapseYesterdayLimitUps ? null : (
                            <div className="border border-slate-100 rounded-xl overflow-hidden shadow-sm">
                              <table className="w-full text-left text-sm">
                                <thead className="bg-white text-horizon-text-secondary font-bold text-[12px] uppercase tracking-wider border-b border-slate-100">
                                  <tr>
                                    <th className="px-6 py-3 w-48">股票</th>
                                    <th className="px-6 py-3 text-center">今日涨幅</th>
                                    <th className="px-6 py-3 text-center">昨日涨幅</th>
                                    <th className="px-6 py-3 text-center">连续&gt;5%</th>
                                  </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-50">
                                  {yesterdayStrongLimitUps.map(stock => (
                                    <tr key={stock.ts_code} className="hover:bg-blue-50/30 transition-colors">
                                      <td className="px-6 py-4 w-48">
                                        <div className="font-bold text-horizon-text-primary whitespace-nowrap overflow-hidden text-ellipsis">{stock.name}</div>
                                        <div className="text-[10px] font-mono text-slate-400 mt-0.5">{stock.ts_code}</div>
                                      </td>
                                      <td className="px-6 py-4 text-center">
                                        <span className={clsx(
                                          "font-bold text-sm",
                                          stock.change_pct >= 0 ? "text-rose-500" : "text-emerald-500"
                                        )}>
                                          {stock.change_pct >= 0 ? '+' : ''}{stock.change_pct.toFixed(2)}%
                                        </span>
                                      </td>
                                      <td className="px-6 py-4 text-center">
                                        <span className="font-bold text-sm text-amber-500">
                                          +{stock.yesterday_change_pct?.toFixed(2)}%
                                        </span>
                                      </td>
                                      <td className="px-6 py-4 text-center">
                                        <DaysBadge days={stock.consecutive_strong_days || 1} />
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          )}
                        </div>
                        {/* 昨日非涨停 */}
                        <div>
                          <button
                            onClick={() => setCollapseYesterdayNonLimit(v => !v)}
                            className="w-full flex items-center justify-between mb-3 text-left"
                          >
                            <h3 className="text-sm font-bold text-amber-600">昨日非涨停（{yesterdayStrongNonLimit.length}）</h3>
                            <span className="text-xs text-slate-500">{collapseYesterdayNonLimit ? '展开' : '收起'}</span>
                          </button>
                          {yesterdayStrongNonLimit.length === 0 ? (
                            <div className="p-6 text-center text-slate-400 border border-slate-100 rounded-xl">无</div>
                          ) : collapseYesterdayNonLimit ? null : (
                            <div className="border border-slate-100 rounded-xl overflow-hidden shadow-sm">
                              <table className="w-full text-left text-sm">
                                <thead className="bg-white text-horizon-text-secondary font-bold text-[12px] uppercase tracking-wider border-b border-slate-100">
                                  <tr>
                                    <th className="px-6 py-3 w-48">股票</th>
                                    <th className="px-6 py-3 text-center">今日涨幅</th>
                                    <th className="px-6 py-3 text-center">昨日涨幅</th>
                                    <th className="px-6 py-3 text-center">连续&gt;5%</th>
                                  </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-50">
                                  {yesterdayStrongNonLimit.map(stock => (
                                    <tr key={stock.ts_code} className="hover:bg-blue-50/30 transition-colors">
                                      <td className="px-6 py-4 w-48">
                                        <div className="font-bold text-horizon-text-primary whitespace-nowrap overflow-hidden text-ellipsis">{stock.name}</div>
                                        <div className="text-[10px] font-mono text-slate-400 mt-0.5">{stock.ts_code}</div>
                                      </td>
                                      <td className="px-6 py-4 text-center">
                                        <span className={clsx(
                                          "font-bold text-sm",
                                          stock.change_pct >= 0 ? "text-rose-500" : "text-emerald-500"
                                        )}>
                                          {stock.change_pct >= 0 ? '+' : ''}{stock.change_pct.toFixed(2)}%
                                        </span>
                                      </td>
                                      <td className="px-6 py-4 text-center">
                                        <span className="font-bold text-sm text-amber-500">
                                          +{stock.yesterday_change_pct?.toFixed(2)}%
                                        </span>
                                      </td>
                                      <td className="px-6 py-4 text-center">
                                        <DaysBadge days={stock.consecutive_strong_days || 1} />
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}
            </>
          )}

          {activeTab === 'tools' && (
            <>
              <div className="flex items-center justify-between">
                <div>
                  <h1 className="text-[24px] font-bold text-horizon-text-primary tracking-tight flex items-center gap-2">
                    <Settings className="text-horizon-brand" />
                    工具
                  </h1>
                  <p className="text-sm text-slate-500 mt-1">运维/数据相关操作和触发器</p>
                </div>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div className="bg-white rounded-[20px] shadow-horizon-card p-6">
                  <h3 className="text-lg font-bold text-horizon-text-primary mb-4">同步市场（Stock Basic）</h3>
                  <button
                    onClick={() => handleToolsSyncMarket()}
                    disabled={syncingMarket}
                    className="px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 disabled:opacity-50 flex items-center gap-2 font-semibold transition-colors"
                  >
                    <RefreshCw size={16} className={syncingMarket ? 'animate-spin' : ''} />
                    {syncingMarket ? '同步中...' : '开始同步'}
                  </button>
                  {syncMarketError && (
                    <div className="mt-3 p-3 bg-rose-50 border border-rose-200 rounded-lg text-rose-600 text-sm">
                      {syncMarketError}
                    </div>
                  )}
                </div>

                <div className="bg-white rounded-[20px] shadow-horizon-card p-6">
                  <h3 className="text-lg font-bold text-horizon-text-primary mb-4">爬取韭研</h3>
                  <button
                    onClick={() => setShowCrawlModal(true)}
                    className="px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 flex items-center gap-2 font-semibold transition-colors"
                  >
                    <Download size={16} />
                    打开爬取对话框
                  </button>
                </div>

                <div className="bg-white rounded-[20px] shadow-horizon-card p-6">
                  <h3 className="text-lg font-bold text-horizon-text-primary mb-4">重建 Relations</h3>
                  <button
                    onClick={() => setShowRebuildModal(true)}
                    className="px-4 py-2 bg-slate-600 text-white rounded-lg hover:bg-slate-700 flex items-center gap-2 font-semibold transition-colors"
                  >
                    <RefreshCw size={16} />
                    打开重建对话框
                  </button>
                </div>

                <div className="bg-white rounded-[20px] shadow-horizon-card p-6">
                  <h3 className="text-lg font-bold text-horizon-text-primary mb-4">Snapshot Trigger</h3>
                  <div className="flex items-end gap-3 flex-wrap">
                    <div>
                      <label className="block text-sm text-slate-500 mb-1">开始日期</label>
                      <input
                        type="date"
                        value={snapshotStartDate}
                        onChange={(e) => setSnapshotStartDate(e.target.value)}
                        className="px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10"
                      />
                    </div>
                    <div>
                      <label className="block text-sm text-slate-500 mb-1">结束日期</label>
                      <input
                        type="date"
                        value={snapshotEndDate}
                        onChange={(e) => setSnapshotEndDate(e.target.value)}
                        className="px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10"
                      />
                    </div>
                    <button
                      onClick={handleTriggerSnapshot}
                      disabled={snapshotLoading}
                      className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2 font-semibold transition-colors"
                    >
                      {snapshotLoading ? (
                        <>
                          <RefreshCw size={16} className="animate-spin" />
                          触发中...
                        </>
                      ) : (
                        '触发'
                      )}
                    </button>
                  </div>
                  {snapshotError && (
                    <div className="mt-3 p-3 bg-rose-50 border border-rose-200 rounded-lg text-rose-600 text-sm">{snapshotError}</div>
                  )}
                  {snapshotSuccessMsg && (
                    <div className="mt-3 p-3 bg-emerald-50 border border-emerald-200 rounded-lg text-emerald-700 text-sm">{snapshotSuccessMsg}</div>
                  )}
                </div>

                {/* Fetch News Test */}
                <div className="bg-white rounded-[20px] shadow-horizon-card p-6">
                  <h3 className="text-lg font-bold text-horizon-text-primary mb-4">热点新闻抓取</h3>
                  <div className="flex items-end gap-3 flex-wrap mb-4">
                    <div>
                      <label className="block text-sm text-slate-500 mb-1">数据源</label>
                      <select
                        value={fetchNewsSource}
                        onChange={(e) => setFetchNewsSource(e.target.value as 'all' | 'cls' | 'eastmoney')}
                        className="px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10"
                      >
                        <option value="all">全部</option>
                        <option value="cls">财联社</option>
                        <option value="eastmoney">东方财富</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm text-slate-500 mb-1">最低重要性</label>
                      <select
                        value={fetchNewsImportance}
                        onChange={(e) => setFetchNewsImportance(e.target.value as 'high' | 'medium' | 'low')}
                        className="px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10"
                      >
                        <option value="high">仅重要 (A级)</option>
                        <option value="medium">较重要+ (B级以上)</option>
                        <option value="low">全部</option>
                      </select>
                    </div>
                    <button
                      onClick={handleFetchNews}
                      disabled={fetchNewsLoading}
                      className="px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700 disabled:opacity-50 flex items-center gap-2 font-semibold transition-colors"
                    >
                      {fetchNewsLoading ? (
                        <>
                          <RefreshCw size={16} className="animate-spin" />
                          抓取中...
                        </>
                      ) : (
                        '抓取新闻'
                      )}
                    </button>
                  </div>
                  {fetchNewsError && (
                    <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-rose-600 text-sm">{fetchNewsError}</div>
                  )}
                  {fetchNewsResult && (
                    <div>
                      <div className="text-sm text-slate-500 mb-2">共 <strong>{fetchNewsResult.total}</strong> 条新闻</div>
                      <div className="max-h-60 overflow-y-auto space-y-2">
                        {fetchNewsResult.news.map((item, i) => (
                          <div key={i} className="p-3 bg-slate-50 rounded-lg border border-slate-100">
                            <div className="flex items-center gap-2 mb-1">
                              <span className={clsx(
                                "text-[10px] font-bold px-1.5 py-0.5 rounded",
                                item.source === 'cls' ? "bg-blue-50 text-blue-600" : "bg-orange-50 text-orange-600"
                              )}>
                                {item.source === 'cls' ? '财联社' : '东方财富'}
                              </span>
                              <span className={clsx(
                                "text-[10px] font-bold px-1.5 py-0.5 rounded",
                                item.importance === 'high' ? "bg-rose-50 text-rose-600" : item.importance === 'medium' ? "bg-amber-50 text-amber-600" : "bg-slate-50 text-slate-400"
                              )}>
                                {item.importance === 'high' ? '重要' : item.importance === 'medium' ? '较重要' : '一般'}
                              </span>
                              <span className="text-xs text-slate-400">{item.published_at}</span>
                            </div>
                            <div className="text-sm font-medium text-slate-700">{item.title}</div>
                            {item.content && <div className="text-xs text-slate-500 mt-1 line-clamp-2">{item.content}</div>}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>

                {/* LLM Classify Test */}
                <div className="bg-white rounded-[20px] shadow-horizon-card p-6 md:col-span-2">
                  <h3 className="text-lg font-bold text-horizon-text-primary mb-2">LLM分类测试（历史数据）</h3>
                  <div className="flex items-end gap-3 flex-wrap mb-4">
                    <div>
                      <label className="block text-sm text-slate-500 mb-1">交易日期</label>
                      <input
                        type="date"
                        value={llmClassifyDate}
                        onChange={(e) => setLlmClassifyDate(e.target.value)}
                        className="px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10"
                      />
                    </div>
                    <button
                      onClick={handleLLMClassifyHistory}
                      disabled={llmClassifyLoading}
                      className="px-4 py-2 bg-violet-600 text-white rounded-lg hover:bg-violet-700 disabled:opacity-50 flex items-center gap-2 font-semibold transition-colors"
                    >
                      {llmClassifyLoading ? (
                        <>
                          <RefreshCw size={16} className="animate-spin" />
                          分类中...
                        </>
                      ) : (
                        '触发LLM分类'
                      )}
                    </button>
                  </div>
                  {llmClassifyError && (
                    <div className="mb-3 p-3 bg-rose-50 border border-rose-200 rounded-lg text-rose-600 text-sm">{llmClassifyError}</div>
                  )}
                  {llmClassifyResult && (
                    <div>
                      <div className="flex items-center gap-4 mb-3 text-sm text-slate-500">
                        <span className="font-medium text-emerald-600">{llmClassifyResult.status === 'completed' ? '分类完成' : llmClassifyResult.status}</span>
                        <span>日期：{llmClassifyResult.date}</span>
                        <span>耗时：{(llmClassifyResult.elapsed_ms / 1000).toFixed(1)}s</span>
                        <span>共 <strong>{llmClassifyResult.total}</strong> 只股票</span>
                      </div>
                      {llmClassifyResult.results.length > 0 && (
                        <div className="overflow-x-auto max-h-80 overflow-y-auto rounded-lg border border-slate-200">
                          <table className="w-full text-left text-sm">
                            <thead className="bg-slate-50 sticky top-0">
                              <tr>
                                <th className="px-3 py-2 text-slate-500 font-medium">代码</th>
                                <th className="px-3 py-2 text-slate-500 font-medium">名称</th>
                                <th className="px-3 py-2 text-slate-500 font-medium">热点</th>
                                <th className="px-3 py-2 text-slate-500 font-medium">TopicID</th>
                                <th className="px-3 py-2 text-slate-500 font-medium">分类</th>
                                <th className="px-3 py-2 text-slate-500 font-medium text-right">置信度</th>
                              </tr>
                            </thead>
                            <tbody>
                              {llmClassifyResult.results.map((item) => (
                                <tr key={item.ts_code} className="border-t border-slate-100 hover:bg-slate-50">
                                  <td className="px-3 py-2 text-slate-500 font-mono text-xs">{item.ts_code}</td>
                                  <td className="px-3 py-2 font-medium">{item.name}</td>
                                  <td className="px-3 py-2">
                                    <span className="bg-violet-100 text-violet-700 text-xs px-2 py-0.5 rounded-full">{item.topic_name}</span>
                                  </td>
                                  <td className="px-3 py-2 text-slate-500 font-mono text-xs">{item.topic_id || <span className="text-rose-400">缺失</span>}</td>
                                  <td className="px-3 py-2">
                                    {item.category ? (
                                      <span className={clsx(
                                        "text-xs px-2 py-0.5 rounded-full",
                                        item.category === '行业' ? "bg-blue-50 text-blue-600" :
                                        item.category === '概念' ? "bg-amber-50 text-amber-600" :
                                        "bg-slate-50 text-slate-500"
                                      )}>{item.category}</span>
                                    ) : (
                                      <span className="text-rose-400 text-xs">缺失</span>
                                    )}
                                  </td>
                                  <td className="px-3 py-2 text-right text-slate-600">{(item.confidence * 100).toFixed(0)}%</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                      {llmClassifyResult.total === 0 && (
                        <p className="text-sm text-slate-400">无符合条件的股票（可能为非交易日）</p>
                      )}
                    </div>
                  )}
                </div>
              </div>
            </>
          )}
          {activeTab === 'stock' && (
            <>
              <div className="flex flex-col gap-6">
                <div className="flex items-center justify-between">
                  <div>
                    <h1 className="text-[24px] font-bold text-horizon-text-primary tracking-tight flex items-center gap-2">
                      <Search className="text-horizon-brand" />
                      股票库查询
                    </h1>
                    <p className="text-sm text-slate-500 mt-1">查询股票基本信息、所属行业及历史关联热点</p>
                  </div>
                  <div className="flex gap-2"></div>
                </div>

                {/* Search & Filters */}
                <div className="bg-white border-none rounded-[20px] p-6 shadow-horizon-card space-y-4">
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
                      <input
                        type="text"
                        placeholder="搜索名称、代码或拼音..."
                        className="w-full bg-slate-50 border border-slate-200 rounded-xl pl-10 pr-4 py-2 text-sm outline-none focus:ring-2 focus:ring-horizon-brand/20 focus:border-horizon-brand transition-all"
                        value={stockQuery}
                        onChange={e => setStockSearchQuery(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && fetchStocks(1)}
                      />
                    </div>
                    <div className="relative z-[60]">
                      <AutocompleteInput
                        value={stockTopicFilter}
                        onChange={setStockTopicFilter}
                        onEnter={() => fetchStocks(1)}
                        placeholder="按热点话题过滤 (如: 华为)..."
                        icon={<BookOpen size={18} />}
                        suggestions={uniqueTopics}
                      />
                    </div>
                    <div className="relative z-[50]">
                      <AutocompleteInput
                        value={stockCategoryFilter}
                        onChange={setStockCategoryFilter}
                        onEnter={() => fetchStocks(1)}
                        placeholder="按分类过滤 (如: 电子)..."
                        icon={<Inbox size={18} />}
                        suggestions={uniqueCategories}
                      />
                    </div>
                  </div>
                  <div className="flex justify-end gap-3">
                    <button
                      onClick={() => fetchStocks(1)}
                      disabled={isSearchingStock}
                      className="flex items-center gap-2 bg-horizon-brand hover:bg-horizon-brand-hover text-white px-6 py-2 rounded-[12px] text-sm font-bold transition-all shadow-md disabled:opacity-50"
                    >
                      <Search size={16} />
                      {isSearchingStock ? '搜索中...' : '开始搜索'}
                    </button>
                  </div>
                  {syncMarketError && (
                    <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-600">
                      {syncMarketError}
                    </div>
                  )}
                </div>

                {/* Results Table */}
                <div className="bg-white border border-slate-200 rounded-2xl shadow-xl shadow-slate-200/50 overflow-hidden">
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                      <thead className="bg-slate-50 text-slate-500 font-bold text-[11px] uppercase tracking-wider border-b border-slate-100">
                        <tr>
                          <th className="px-6 py-4">股票名称</th>
                          <th className="px-6 py-4">代码</th>
                          <th className="px-6 py-4">行业</th>
                          <th className="px-6 py-4">板块</th>
                          <th className="px-6 py-4 text-right">操作</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-50">
                        {stockResults.length === 0 ? (
                          <tr>
                            <td colSpan={5} className="px-6 py-12 text-center text-slate-400 italic">
                              {isSearchingStock ? '正在加载数据...' : '未找到匹配的股票数据'}
                            </td>
                          </tr>
                        ) : (
                          stockResults.map(s => (
                            <tr key={s.ts_code} className="hover:bg-slate-50 transition-colors group">
                              <td className="px-6 py-4">
                                <span className="font-bold text-horizon-text-primary">{s.name}</span>
                                {s.is_st && <span className="ml-2 px-1.5 py-0.5 bg-rose-50 text-rose-600 text-[10px] font-bold rounded border border-rose-100">ST</span>}
                              </td>
                              <td className="px-6 py-4 font-mono text-slate-500">{s.ts_code}</td>
                              <td className="px-6 py-4 text-slate-600">{s.industry || '-'}</td>
                              <td className="px-6 py-4 text-slate-500">
                                {(() => {
                                  if (!s.board_code) return '-';
                                  if (s.board_code.toLowerCase() === 'main') return '主板';
                                  if (s.board_code.toLowerCase() === 'gem') return '创业板';
                                  if (s.board_code.toLowerCase() === 'star') return '科创板';
                                  if (s.board_code.toLowerCase() === 'bse') return '北交所';
                                  return s.board_code;
                                })()}
                              </td>
                              <td className="px-6 py-4 text-right">
                                <button
                                  onClick={() => fetchStockDetail(s.ts_code)}
                                  className="text-blue-600 hover:text-blue-800 font-bold text-xs"
                                >
                                  查看关联热点
                                </button>
                                <button
                                  onClick={() => openTopicManager({ ts_code: s.ts_code, name: s.name })}
                                  className="text-emerald-600 hover:text-emerald-800 font-bold text-xs ml-3"
                                >
                                  修改 topic
                                </button>
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>

                  {/* Pagination Controls */}
                  {stockTotal > stockPageSize && (
                    <div className="px-6 py-4 bg-slate-50/50 border-t border-slate-100 flex items-center justify-between">
                      <div className="text-xs text-slate-500">
                        共 <span className="font-bold text-slate-700">{stockTotal}</span> 条结果，当前第 {stockPage} 页
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          disabled={stockPage === 1 || isSearchingStock}
                          onClick={() => fetchStocks(stockPage - 1)}
                          className="px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs font-bold text-slate-600 hover:bg-slate-50 disabled:opacity-30 transition-all"
                        >
                          上一页
                        </button>
                        <button
                          disabled={stockPage * stockPageSize >= stockTotal || isSearchingStock}
                          onClick={() => fetchStocks(stockPage + 1)}
                          className="px-3 py-1.5 bg-white border border-slate-200 rounded-lg text-xs font-bold text-slate-600 hover:bg-slate-50 disabled:opacity-30 transition-all"
                        >
                          下一页
                        </button>
                      </div>
                    </div>
                  )}
                </div>

                {/* Detail View (Modal or Section) */}
                {selectedStock && (
                  <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-[60] flex items-center justify-center p-4">
                    <div className="bg-white rounded-[20px] w-full max-w-4xl max-h-[90vh] overflow-hidden shadow-horizon-card animate-in zoom-in-95 duration-200 flex flex-col">
                      <div className="p-6 border-none flex items-center justify-between bg-white">
                        <div className="flex items-center gap-4">
                          <div className="w-12 h-12 bg-horizon-brand rounded-2xl flex items-center justify-center text-white shadow-lg shadow-horizon-brand/20">
                            <BarChart3 size={24} />
                          </div>
                          <div>
                            <h3 className="text-[20px] font-bold text-horizon-text-primary">{selectedStock.info.name}</h3>
                            <p className="text-xs font-mono text-slate-500 tracking-wider">{selectedStock.info.ts_code}</p>
                          </div>
                          <button
                            onClick={() => openTopicManager({ ts_code: selectedStock.info.ts_code, name: selectedStock.info.name }, selectedStock.topics)}
                            className="px-3 py-1.5 bg-emerald-50 text-emerald-700 rounded-lg text-xs font-bold hover:bg-emerald-100 transition-colors"
                          >
                            修改 topic
                          </button>
                        </div>
                        <button
                          onClick={() => setSelectedStock(null)}
                          className="p-2 hover:bg-slate-200 rounded-xl transition-all text-slate-400"
                        >
                          <X size={24} />
                        </button>
                      </div>

                      <div className="flex-1 overflow-y-auto p-6 space-y-6">
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                          <div className="bg-slate-50 p-4 rounded-2xl border border-slate-100">
                            <div className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-1">所属行业</div>
                            <div className="font-bold text-slate-700">{selectedStock.info.industry || '-'}</div>
                          </div>
                          <div className="bg-slate-50 p-4 rounded-2xl border border-slate-100">
                            <div className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-1">所属板块</div>
                            <div className="font-bold text-slate-700">
                              {(() => {
                                if (!selectedStock.info.board_code) return '-';
                                if (selectedStock.info.board_code.toLowerCase() === 'main') return '主板';
                                if (selectedStock.info.board_code.toLowerCase() === 'gem') return '创业板';
                                if (selectedStock.info.board_code.toLowerCase() === 'star') return '科创板';
                                if (selectedStock.info.board_code.toLowerCase() === 'bse') return '北交所';
                                return selectedStock.info.board_code;
                              })()}
                            </div>
                          </div>
                          <div className="bg-slate-50 p-4 rounded-2xl border border-slate-100">
                            <div className="text-[10px] font-bold text-slate-400 uppercase tracking-widest mb-1">股票状态</div>
                            <div>
                              <span className={clsx(
                                "px-2 py-0.5 rounded text-[10px] font-bold",
                                selectedStock.info.is_st ? "bg-rose-50 text-rose-600" : "bg-emerald-50 text-emerald-600"
                              )}>
                                {selectedStock.info.is_st ? "ST" : "正常交易"}
                              </span>
                            </div>
                          </div>
                        </div>

                        <div>
                          <h4 className="font-bold text-slate-800 mb-4 flex items-center gap-2">
                            <BookOpen size={18} className="text-blue-600" />
                            关联热点话题记录
                          </h4>
                          <div className="border border-slate-100 rounded-2xl overflow-hidden">
                            <table className="w-full text-left text-base">
                              <thead className="bg-slate-50 text-slate-500 font-bold text-[11px] uppercase tracking-wider">
                                <tr>
                                  <th className="px-6 py-4">热点名称</th>
                                  <th className="px-6 py-4">所属分类</th>
                                  <th className="px-6 py-4 text-center">命中次数</th>
                                  <th className="px-6 py-4 text-right">最后出现</th>
                                </tr>
                              </thead>
                              <tbody className="divide-y divide-slate-100">
                                {selectedStock.topics.length === 0 ? (
                                  <tr>
                                    <td colSpan={4} className="px-6 py-12 text-center text-slate-400 italic text-base">
                                      暂无关联热点话题数据
                                    </td>
                                  </tr>
                                ) : (
                                  selectedStock.topics
                                    .sort((a, b) => b.hit_count - a.hit_count)
                                    .map(t => (
                                      <tr key={t.topic_id} className="hover:bg-slate-50 transition-colors">
                                        <td className="px-6 py-4 font-bold text-slate-700">{t.topic_name}</td>
                                        <td className="px-6 py-4">
                                          <span className="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-[10px] font-bold">
                                            {t.category}
                                          </span>
                                        </td>
                                        <td className="px-6 py-4 text-center font-mono text-slate-600">{t.hit_count}</td>
                                        <td className="px-6 py-4 text-right text-slate-500">
                                          {t.last_seen_date ? new Date(t.last_seen_date).toLocaleDateString() : '-'}
                                        </td>
                                      </tr>
                                    ))
                                )}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </>
          )}

          {activeTab === 'dictionary' && (
            <>
              {/* Dictionary Header */}
              <div className="flex items-center justify-between">
                <div>
                  <h1 className="text-[24px] font-bold text-horizon-text-primary tracking-tight flex items-center gap-2">
                    <BookOpen className="text-horizon-brand" />
                    主题词典管理
                  </h1>
                  <p className="text-sm text-slate-500 mt-1">管理原始主题名到标准名称的映射关系</p>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => setHiddenDictIds([])}
                    disabled={hiddenDictIds.length === 0}
                    className="flex items-center gap-2 bg-white border border-slate-200 hover:border-slate-300 text-slate-600 disabled:text-slate-300 disabled:border-slate-100 px-4 py-2 rounded-[12px] text-sm font-semibold transition-all disabled:cursor-not-allowed"
                  >
                    <Eye size={16} />
                    全部取消隐藏
                  </button>
                  <button
                    onClick={() => setIsAddingDict(true)}
                    className="flex items-center gap-2 bg-horizon-brand hover:bg-horizon-brand-hover text-white px-4 py-2 rounded-[12px] text-sm font-semibold shadow-md transition-all"
                  >
                    <Plus size={16} />
                    新增词条
                  </button>
                </div>
              </div>

              {/* Dictionary Table */}
              <div className="bg-white border-none rounded-[20px] shadow-horizon-card overflow-hidden">
                <div className="p-6 border-none bg-white">
                  <div className="relative w-72">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={16} />
                    <input
                      type="text"
                      placeholder="搜索原始名或标准名..."
                      value={dictSearch}
                      onChange={e => setDictSearch(e.target.value)}
                      className="bg-white border border-slate-200 rounded-xl pl-10 pr-4 py-2 text-sm outline-none focus:ring-2 focus:ring-horizon-brand/20 focus:border-horizon-brand transition-all w-full"
                    />
                  </div>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-white text-horizon-text-secondary font-bold text-[12px] uppercase tracking-wider border-b border-slate-100">
                      <tr>
                        <th className="px-6 py-4">原始主题名 (Raw)</th>
                        <th className="px-6 py-4">标准名称 (Normalized)</th>
                        <th className="px-6 py-4">所属分类 (Category)</th>
                        <th className="px-6 py-4 text-right">操作</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {isAddingDict && (
                        <AddDictRow
                          onSave={handleAddDict}
                          onCancel={() => setIsAddingDict(false)}
                        />
                      )}
                      {filteredDictionary.map(d => (
                        <tr key={d.id} className="hover:bg-slate-50 transition-colors group">
                          <td className="px-6 py-4 font-medium text-slate-700">
                            {editingDictId === d.id ? (
                              <input
                                className="w-full bg-white border border-blue-200 rounded px-2 py-1 outline-none"
                                value={newDict.raw_topic_name}
                                onChange={e => setNewDict({ ...newDict, raw_topic_name: e.target.value })}
                              />
                            ) : d.raw_topic_name}
                          </td>
                          <td className="px-6 py-4 text-slate-600">
                            {editingDictId === d.id ? (
                              <input
                                className="w-full bg-white border border-blue-200 rounded px-2 py-1 outline-none"
                                value={newDict.normalized_name}
                                onChange={e => setNewDict({ ...newDict, normalized_name: e.target.value })}
                              />
                            ) : (
                              <span className="px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs font-bold border border-blue-100">
                                {d.normalized_name}
                              </span>
                            )}
                          </td>
                          <td className="px-6 py-4 text-slate-500">
                            {editingDictId === d.id ? (
                              <input
                                className="w-full bg-white border border-blue-200 rounded px-2 py-1 outline-none"
                                value={newDict.category}
                                onChange={e => setNewDict({ ...newDict, category: e.target.value })}
                              />
                            ) : d.category}
                          </td>
                          <td className="px-6 py-4 text-right space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            {editingDictId === d.id ? (
                              <>
                                <button onClick={() => handleUpdateDict(d.id)} className="text-blue-600 hover:text-blue-800 p-1.5 rounded-lg hover:bg-blue-100">
                                  <Check size={18} />
                                </button>
                                <button onClick={() => setEditingDictId(null)} className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg hover:bg-slate-200">
                                  <X size={18} />
                                </button>
                              </>
                            ) : (
                              <>
                                <button
                                  onClick={() => {
                                    setEditingDictId(d.id);
                                    setNewDict(d);
                                  }}
                                  className="text-slate-400 hover:text-blue-600 p-1.5 rounded-lg hover:bg-blue-50 transition-colors"
                                >
                                  <Edit2 size={16} />
                                </button>
                                <button
                                  onClick={() => setHiddenDictIds(prev => prev.includes(d.id) ? prev : [...prev, d.id])}
                                  className="text-slate-400 hover:text-amber-600 p-1.5 rounded-lg hover:bg-amber-50 transition-colors"
                                  title="隐藏此行"
                                >
                                  <EyeOff size={16} />
                                </button>
                                <button
                                  onClick={() => handleDeleteDict(d.id)}
                                  className="text-slate-400 hover:text-rose-600 p-1.5 rounded-lg hover:bg-rose-50 transition-colors"
                                >
                                  <Trash2 size={16} />
                                </button>
                              </>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      <TopicRelationManagerModal
        open={Boolean(topicManagerStock)}
        stock={topicManagerStock}
        initialRelations={topicManagerRelations}
        topics={allTopicsForPicker}
        onClose={() => setTopicManagerStock(null)}
        onUpdated={handleTopicRelationsUpdated}
      />

      {/* Rebuild Modal */}
      {showRebuildModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-[20px] p-6 w-[500px] max-h-[80vh] overflow-y-auto shadow-2xl">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-bold text-horizon-text-primary">重建 Topic Relations</h3>
              <button onClick={() => setShowRebuildModal(false)} className="text-slate-400 hover:text-slate-600">
                <X size={20} />
              </button>
            </div>

            <div className="space-y-4">
              <div className="flex gap-4">
                <div className="flex-1">
                  <label className="block text-sm text-slate-500 mb-1">开始日期</label>
                  <input
                    type="date"
                    value={rebuildStartDate}
                    onChange={(e) => setRebuildStartDate(e.target.value)}
                    className="w-full px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10"
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-sm text-slate-500 mb-1">结束日期</label>
                  <input
                    type="date"
                    value={rebuildEndDate}
                    onChange={(e) => setRebuildEndDate(e.target.value)}
                    className="w-full px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10"
                  />
                </div>
              </div>

              <button
                onClick={handleRebuild}
                disabled={rebuildLoading}
                className="w-full py-2.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center justify-center gap-2 font-semibold transition-colors"
              >
                {rebuildLoading ? (
                  <>
                    <RefreshCw size={16} className="animate-spin" />
                    处理中...
                  </>
                ) : (
                  '开始重建'
                )}
              </button>

              {rebuildError && (
                <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-rose-600 text-sm">
                  {rebuildError}
                </div>
              )}

              {rebuildResult && (
                <div className="space-y-3">
                  <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-lg">
                    <div className="text-emerald-700 font-semibold">重建成功</div>
                    <div className="text-sm text-slate-600 mt-1">
                      日期范围: {rebuildResult.start_date} ~ {rebuildResult.end_date}
                    </div>
                    <div className="text-sm text-slate-600">
                      Topics: {rebuildResult.topics_count} | Relations: {rebuildResult.relations_count}
                    </div>
                  </div>

                  {rebuildResult.topics.length > 0 && (
                    <div className="border border-slate-200 rounded-lg overflow-hidden">
                      <table className="w-full text-sm">
                        <thead className="bg-slate-50">
                          <tr>
                            <th className="px-3 py-2 text-left text-slate-600 font-semibold">原始名称</th>
                            <th className="px-3 py-2 text-left text-slate-600 font-semibold">归一化名称</th>
                            <th className="px-3 py-2 text-right text-slate-600 font-semibold">股票数</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                          {rebuildResult.topics.map((t, i) => (
                            <tr key={i}>
                              <td className="px-3 py-2 text-slate-700">{t.name}</td>
                              <td className="px-3 py-2 text-slate-700">{t.normalized_name}</td>
                              <td className="px-3 py-2 text-right text-slate-700">{t.stocks_count}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Crawl Jiuyan Modal */}
      {showCrawlModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-[20px] p-6 w-[600px] max-h-[80vh] overflow-y-auto shadow-2xl">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-bold text-horizon-text-primary">爬取韭研数据</h3>
              <button onClick={() => setShowCrawlModal(false)} className="text-slate-400 hover:text-slate-600">
                <X size={20} />
              </button>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-slate-500 mb-1">粘贴 curl 命令</label>
                <textarea
                  value={curlInput}
                  onChange={(e) => setCurlInput(e.target.value)}
                  placeholder="curl 'https://app.jiuyangongshe.com/api/v1/...' -H 'token: ...' -H 'timestamp: ...' -b '...' --data-raw '{...}'"
                  className="w-full h-40 px-3 py-2 bg-white rounded-lg border border-slate-200 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/10 font-mono text-xs resize-none"
                />
                <p className="text-xs text-slate-400 mt-1">从浏览器开发者工具复制 curl 命令，需包含 token、timestamp、cookie 和 date</p>
              </div>

              <button
                onClick={handleCrawlJiuyan}
                disabled={crawlLoading}
                className="w-full py-2.5 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 disabled:opacity-50 flex items-center justify-center gap-2 font-semibold transition-colors"
              >
                {crawlLoading ? (
                  <>
                    <RefreshCw size={16} className="animate-spin" />
                    爬取中...
                  </>
                ) : (
                  '开始爬取'
                )}
              </button>

              {crawlError && (
                <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-rose-600 text-sm">
                  {crawlError}
                </div>
              )}

              {crawlResult && (
                <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-lg">
                  <div className="text-emerald-700 font-semibold">爬取成功</div>
                  <div className="text-sm text-slate-600 mt-1">
                    日期: {crawlResult.date}
                  </div>
                  <div className="text-sm text-slate-600">
                    Topics: {crawlResult.topics_count} | Stocks: {crawlResult.stocks_count}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Missing Topics Modal */}
      {showMissingTopicsModal && crawlMissingTopics.length > 0 && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-[20px] p-6 w-[400px] shadow-2xl">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-bold text-rose-600">缺失 Topic 映射</h3>
              <button onClick={() => setShowMissingTopicsModal(false)} className="text-slate-400 hover:text-slate-600">
                <X size={20} />
              </button>
            </div>

            <p className="text-sm text-slate-600 mb-3">
              以下 topics 在 topic_dictionary 中没有映射，请先添加：
            </p>

            <div className="max-h-48 overflow-y-auto mb-4">
              <ul className="space-y-1">
                {crawlMissingTopics.map((t, i) => (
                  <li key={i} className="px-3 py-1.5 bg-slate-100 rounded-lg text-sm text-slate-700">{t}</li>
                ))}
              </ul>
            </div>

            <div className="flex gap-3">
              <button
                onClick={() => {
                  setShowMissingTopicsModal(false);
                  setActiveTab('dictionary');
                }}
                className="flex-1 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold transition-colors"
              >
                去添加映射
              </button>
              <button
                onClick={() => {
                  setShowMissingTopicsModal(false);
                  setRebuildStartDate(crawlFailedDate);
                  setRebuildEndDate(crawlFailedDate);
                  setShowRebuildModal(true);
                  setActiveTab('dictionary');
                }}
                className="flex-1 py-2 bg-slate-200 text-slate-700 rounded-lg hover:bg-slate-300 font-semibold transition-colors"
              >
                去重建
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
