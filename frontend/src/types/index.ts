export interface TopicRelation {
  topic_id: number;
  topic_name: string;
  category: string;
  source: string;
  hit_count: number;
  last_seen_date?: string;
  first_seen_date?: string;
  updated_at?: string;
  confidence: number;
}

export interface ReclassifyResult {
  ts_code: string;
  name: string;
  price: number;
  pre_close: number;
  change_pct: number;
  pool_type: number;
  limit_up_price?: number;
  is_limit_up: boolean;
  is_above_5pct: boolean;
  board_code: string;
  topics?: TopicRelation[];
  all_topics?: TopicRelation[];
  first_time?: string;
  last_time?: string;
  limit_times?: number;
  limit_times_display?: string;
  yesterday_change_pct?: number;
  is_yesterday_strong?: boolean;
  consecutive_strong_days?: number;
  total_mv?: number;
  vol?: number;
  amount?: number;
}

export interface PoolData {
  name: string;
  categories: Record<string, Record<string, ReclassifyResult[]>>;
  total: number;
}

export interface ProcessedData {
  pools: {
    1: PoolData;
    2: PoolData;
  };
  limitUpTopics: Set<string>;
  limitUpCategories: Set<string>;
}
