import { useEffect, useMemo, useState } from 'react';
import axios from 'axios';
import { BookOpen, RefreshCw, Search, Trash2, X } from 'lucide-react';
import type { TopicRelation } from '../../types';

type TopicOption = {
  id: number;
  name: string;
  category: string;
};

type TopicRelationManagerModalProps = {
  open: boolean;
  stock: { ts_code: string; name: string } | null;
  initialRelations: TopicRelation[];
  topics: TopicOption[];
  onClose: () => void;
  onUpdated: (relations: TopicRelation[]) => Promise<void>;
};

function formatDate(date?: string) {
  if (!date) return '-';
  const parsed = new Date(date);
  if (Number.isNaN(parsed.getTime())) return date;
  return parsed.toLocaleDateString();
}

export function TopicRelationManagerModal({
  open,
  stock,
  initialRelations,
  topics,
  onClose,
  onUpdated,
}: TopicRelationManagerModalProps) {
  const [keyword, setKeyword] = useState('');
  const [selectedTopicId, setSelectedTopicId] = useState<number | ''>('');
  const [relations, setRelations] = useState<TopicRelation[]>(initialRelations);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setRelations(initialRelations);
    setKeyword('');
    setSelectedTopicId('');
    setError(null);
  }, [open, initialRelations]);

  const filteredTopics = useMemo(() => {
    const normalized = keyword.trim().toLowerCase();
    if (!normalized) return topics.slice(0, 50);
    return topics
      .filter((topic) =>
        topic.name.toLowerCase().includes(normalized) ||
        topic.category.toLowerCase().includes(normalized)
      )
      .slice(0, 50);
  }, [keyword, topics]);

  if (!open || !stock) return null;

  const handleAdd = async () => {
    if (!selectedTopicId) {
      setError('请选择 topic');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const res = await axios.post(`/api/v1/stocks/${stock.ts_code}/topic-relations/manual`, {
        topic_id: selectedTopicId,
      }, {
        headers: { 'X-API-Key': 'test-api-key' },
      });
      const nextRelations = res.data?.data || [];
      setRelations(nextRelations);
      await onUpdated(nextRelations);
      setSelectedTopicId('');
      setKeyword('');
    } catch (err) {
      console.error(err);
      setError('新增 manual topic 失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (topicId: number) => {
    setSubmitting(true);
    setError(null);
    try {
      const res = await axios.delete(`/api/v1/stocks/${stock.ts_code}/topic-relations/${topicId}`, {
        headers: { 'X-API-Key': 'test-api-key' },
      });
      const nextRelations = res.data?.data || [];
      setRelations(nextRelations);
      await onUpdated(nextRelations);
    } catch (err) {
      console.error(err);
      setError('删除 topic relation 失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-[70] flex items-center justify-center p-4">
      <div className="bg-white rounded-[20px] w-full max-w-4xl max-h-[90vh] overflow-hidden shadow-horizon-card animate-in zoom-in-95 duration-200 flex flex-col">
        <div className="p-6 border-b border-slate-100 flex items-center justify-between bg-white">
          <div>
            <h3 className="text-[20px] font-bold text-horizon-text-primary flex items-center gap-2">
              <BookOpen size={20} className="text-emerald-600" />
              修改 topic
            </h3>
            <p className="text-xs font-mono text-slate-500 mt-1">{stock.name} · {stock.ts_code}</p>
          </div>
          <button
            onClick={onClose}
            className="p-2 hover:bg-slate-200 rounded-xl transition-all text-slate-400"
            disabled={submitting}
          >
            <X size={24} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          <div className="bg-slate-50 p-4 rounded-2xl border border-slate-100 space-y-4">
            <div className="flex items-center gap-2 text-sm font-bold text-slate-700">
              <Search size={16} className="text-slate-400" />
              新增 manual topic
            </div>
            <div className="grid grid-cols-1 md:grid-cols-[1fr_1fr_auto] gap-3">
              <input
                type="text"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                placeholder="搜索 topic 名称或分类"
                className="w-full bg-white border border-slate-200 rounded-xl px-4 py-2 text-sm outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all"
                disabled={submitting}
              />
              <select
                value={selectedTopicId}
                onChange={(e) => setSelectedTopicId(e.target.value ? Number(e.target.value) : '')}
                className="w-full bg-white border border-slate-200 rounded-xl px-4 py-2 text-sm outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500 transition-all"
                disabled={submitting}
              >
                <option value="">请选择 topic</option>
                {filteredTopics.map((topic) => (
                  <option key={topic.id} value={topic.id}>
                    {topic.name} / {topic.category}
                  </option>
                ))}
              </select>
              <button
                onClick={handleAdd}
                disabled={submitting || !selectedTopicId}
                className="px-4 py-2 bg-emerald-600 text-white rounded-xl hover:bg-emerald-700 disabled:opacity-50 font-semibold transition-colors whitespace-nowrap"
              >
                {submitting ? <RefreshCw size={16} className="animate-spin inline" /> : '新增为人工 topic'}
              </button>
            </div>
            {error && (
              <div className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-600">
                {error}
              </div>
            )}
          </div>

          <div>
            <h4 className="font-bold text-slate-800 mb-4">当前 relation 列表</h4>
            <div className="border border-slate-100 rounded-2xl overflow-hidden">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-slate-500 font-bold text-[11px] uppercase tracking-wider">
                  <tr>
                    <th className="px-4 py-3">热点名称</th>
                    <th className="px-4 py-3">分类</th>
                    <th className="px-4 py-3">来源</th>
                    <th className="px-4 py-3 text-center">命中次数</th>
                    <th className="px-4 py-3 text-right">最后出现</th>
                    <th className="px-4 py-3 text-right">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {relations.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-6 py-10 text-center text-slate-400 italic">
                        暂无 relation
                      </td>
                    </tr>
                  ) : (
                    relations.map((relation) => (
                      <tr key={relation.topic_id} className="hover:bg-slate-50 transition-colors">
                        <td className="px-4 py-3 font-semibold text-slate-700">{relation.topic_name}</td>
                        <td className="px-4 py-3 text-slate-600">{relation.category || '-'}</td>
                        <td className="px-4 py-3 text-slate-600">{relation.source}</td>
                        <td className="px-4 py-3 text-center text-slate-600">{relation.hit_count}</td>
                        <td className="px-4 py-3 text-right text-slate-500">{formatDate(relation.last_seen_date)}</td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={() => handleDelete(relation.topic_id)}
                            disabled={submitting}
                            className="inline-flex items-center gap-1 text-rose-600 hover:text-rose-700 disabled:opacity-50 text-xs font-bold"
                          >
                            <Trash2 size={14} />
                            删除
                          </button>
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
  );
}
