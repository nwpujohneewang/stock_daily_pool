package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"stock/config"
	"stock/dal/db"
	redis2 "stock/dal/redis"
	"stock/internal/external/tushare"
	"stock/internal/pkg/attribution"
	"stock/internal/pkg/limiter"
	"stock/internal/pkg/logger"
	"stock/internal/service"
	"stock/model/dal_model"
)

type relationInfo struct {
	TopicID      int64
	TopicName    string
	Category     string
	HitCount     int
	LastSeenDate string
	Source       string
	Confidence   float64
}

type attributionInfo struct {
	TopicID      int64
	TopicName    string
	Category     string
	HitCount     int
	LastSeenDate string
	Source       string
	Score        float64
}

type stockInfo struct {
	tsCode string
	name   string
	pct    float64
	vol    int64
	amount float64
	attr   attributionInfo
	rels   []relationInfo
}

type topicData struct {
	meta   attributionInfo
	stocks []stockInfo
}

type poolData struct {
	name         string
	total        int
	topics       map[string]topicData
	unclassified []stockInfo
}

var htmlTemplate = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>股票分类结果 {{.Date}}</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f1117; color: #e5e7eb; min-height: 100vh; padding: 24px; }

.header { max-width: 1400px; margin: 0 auto 20px; }
.header h1 { font-size: 20px; font-weight: 600; color: #f9fafb; margin-bottom: 8px; }
.header .meta { font-size: 13px; color: #6b7280; }

.summary-bar { max-width: 1400px; margin: 0 auto 24px; display: flex; gap: 16px; flex-wrap: wrap; }
.summary-item { background: #1f2937; border: 1px solid #374151; border-radius: 8px; padding: 12px 20px; min-width: 120px; }
.summary-item .label { font-size: 12px; color: #6b7280; margin-bottom: 4px; }
.summary-item .value { font-size: 22px; font-weight: 700; }
.summary-item .value.red { color: #ef4444; }
.summary-item .value.orange { color: #f97316; }
.summary-item .value.gray { color: #9ca3af; }

.pool-container { max-width: 1400px; margin: 0 auto; }

.pool-section { background: #1f2937; border: 1px solid #374151; border-radius: 12px; margin-bottom: 24px; overflow: hidden; }
.pool-section.hidden { display: none; }

.pool-header { padding: 16px 20px; border-bottom: 1px solid #374151; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; }
.pool-title { font-size: 15px; font-weight: 600; color: #f9fafb; display: flex; align-items: center; gap: 8px; }
.pool-title .badge { background: #374151; color: #9ca3af; font-size: 12px; font-weight: 500; padding: 2px 8px; border-radius: 10px; }
.pool-header .controls { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }

.search-input { background: #111827; border: 1px solid #374151; color: #e5e7eb; padding: 6px 12px; border-radius: 6px; font-size: 13px; width: 200px; outline: none; }
.search-input:focus { border-color: #3b82f6; }
.search-input::placeholder { color: #6b7280; }

.topic-filter-btn { background: #374151; color: #d1d5db; border: 1px solid #4b5563; padding: 4px 10px; border-radius: 6px; font-size: 12px; cursor: pointer; transition: all 0.15s; }
.topic-filter-btn:hover { background: #4b5563; color: #f9fafb; }
.topic-filter-btn.active { background: #3b82f6; color: #fff; border-color: #3b82f6; }
.topic-filter-btn.all { background: #6b7280; color: #e5e7eb; border-color: #6b7280; }

.topic-section { border-bottom: 1px solid #1f2937; }
.topic-section:last-child { border-bottom: none; }
.topic-section.hidden { display: none; }

.topic-header { padding: 10px 20px; background: #111827; cursor: pointer; display: flex; align-items: center; justify-content: space-between; user-select: none; }
.topic-header:hover { background: #1a2234; }
.topic-name { font-size: 13px; font-weight: 600; color: #f3f4f6; display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.topic-name .count { background: #374151; color: #9ca3af; font-size: 11px; font-weight: 500; padding: 1px 6px; border-radius: 8px; }
.topic-name .collapse-icon { font-size: 10px; color: #6b7280; transition: transform 0.2s; }
.topic-section.collapsed .collapse-icon { transform: rotate(-90deg); }
.topic-section.collapsed .topic-body { display: none; }

.topic-body { padding: 0; }

.attr-meta { padding: 6px 20px; background: #1a2234; border-bottom: 1px solid #2d3748; display: flex; gap: 16px; flex-wrap: wrap; align-items: center; }
.attr-meta .attr-item { font-size: 11px; color: #9ca3af; display: flex; gap: 4px; }
.attr-meta .attr-item span { color: #e5e7eb; font-weight: 600; }
.attr-meta .attr-item.source span { color: #a78bfa; }

.topic-table { display: table; width: 100%; border-collapse: collapse; }
.table-head { display: table-row; background: #1a2234; }
.table-head th { padding: 8px 20px; text-align: left; font-size: 11px; font-weight: 600; color: #6b7280; text-transform: uppercase; letter-spacing: 0.05em; border-bottom: 1px solid #374151; white-space: nowrap; }
.table-row { display: table-row; transition: background 0.1s; }
.table-row:hover { background: #1f2937; }
.table-row td { padding: 9px 20px; font-size: 13px; border-bottom: 1px solid #1f2937; vertical-align: middle; white-space: nowrap; }
.table-row:last-child td { border-bottom: none; }
.ts-code { color: #6b7280; font-size: 12px; font-family: "SF Mono", Consolas, monospace; }
.stock-name { color: #e5e7eb; font-weight: 500; display: flex; align-items: center; gap: 6px; }
.pct { font-weight: 600; }
.pct.up { color: #ef4444; }
.pct.up5 { color: #f97316; }
.num { color: #d1d5db; font-size: 12px; }

.stock-toggle { cursor: pointer; color: #6b7280; font-size: 10px; padding: 2px 6px; border-radius: 4px; background: #374151; border: none; transition: all 0.15s; white-space: nowrap; user-select: none; }
.stock-toggle:hover { background: #4b5563; color: #e5e7eb; }
.stock-toggle.open { background: #3b82f6; color: #fff; }

.stock-detail { display: none; background: #161b22; border-bottom: 1px solid #1f2937; }
.stock-detail.open { display: table-row; }
.stock-detail td { padding: 0 20px 12px 44px; vertical-align: top; }

.rel-list { display: flex; flex-wrap: wrap; gap: 8px; padding: 8px 0; }
.rel-item { display: inline-flex; align-items: center; gap: 8px; background: #1f2937; border: 1px solid #374151; border-radius: 6px; padding: 4px 10px; font-size: 12px; }
.rel-name { color: #93c5fd; font-weight: 500; }
.rel-hit { color: #fbbf24; font-size: 11px; }
.rel-date { color: #6b7280; font-size: 11px; }

.empty-state { padding: 40px; text-align: center; color: #6b7280; font-size: 14px; }

.tab-bar { max-width: 1400px; margin: 0 auto 16px; display: flex; gap: 4px; }
.tab { background: #1f2937; border: 1px solid #374151; color: #6b7280; padding: 8px 20px; border-radius: 8px; cursor: pointer; font-size: 14px; font-weight: 500; transition: all 0.15s; }
.tab:hover { color: #e5e7eb; }
.tab.active { background: #374151; color: #f9fafb; border-color: #4b5563; }
.tab .tab-count { margin-left: 6px; background: #374151; color: #9ca3af; font-size: 12px; padding: 1px 6px; border-radius: 8px; }
.tab.active .tab-count { background: #4b5563; color: #e5e7eb; }
.tab.uc { border-color: #4b5563; }
.tab.uc.active { background: #374151; }
</style>
</head>
<body>

<div class="header">
  <h1>📊 股票分类结果 — {{.Date}}</h1>
  <div class="meta">归属模式: {{.Mode}} &nbsp;|&nbsp; 生成时间: {{.GenTime}}</div>
</div>

<div class="summary-bar">
  <div class="summary-item">
    <div class="label">涨停池</div>
    <div class="value red">{{.LimitUpTotal}}</div>
  </div>
  <div class="summary-item">
    <div class="label">5%池</div>
    <div class="value orange">{{.Above5Total}}</div>
  </div>
  <div class="summary-item">
    <div class="label">未分类</div>
    <div class="value gray">{{.UnclassifiedTotal}}</div>
  </div>
  <div class="summary-item">
    <div class="label">热点数</div>
    <div class="value">{{.TopicCount}}</div>
  </div>
</div>

<div class="tab-bar" id="tabBar"></div>
<div id="poolContainer"></div>

<script>
{{.Script}}
</script>
</body>
</html>`

func main() {
	date := flag.String("date", "2026-03-20", "trade date")
	mode := flag.String("mode", "recent", "attribution mode: normal, concept, recent")
	outPath := flag.String("out", "", "output HTML path (default: reclassify_{date}.html)")
	flag.Parse()

	if *date == "" {
		fmt.Println("Usage: go run cmd/reclassify/main.go -date 2026-03-18 [-mode recent] [-out path]")
		os.Exit(1)
	}
	if *outPath == "" {
		*outPath = fmt.Sprintf("reclassify_%s.html", strings.ReplaceAll(*date, "-", ""))
	}

	var classifySvc service.ClassifyServiceInterface
	switch *mode {
	case "recent":
		classifySvc = service.NewClassifyServiceWithMode(attribution.WeightModeRecent)
	default:
		classifySvc = service.NewClassifyService()
	}

	ctx := context.Background()

	if err := config.Init("config/config.yaml"); err != nil {
		log.Fatalf("init config: %v", err)
	}
	if err := logger.Init("debug", "console", "stdout"); err != nil {
		log.Fatalf("init logger: %v", err)
	}
	db.Init()
	redis2.Init()

	tradeDate := strings.ReplaceAll(*date, "-", "")
	tushareClient := tushare.NewClient(&config.GlobalConfig.Tushare, config.GlobalConfig.Retry)

	fmt.Printf("Fetching daily data for %s...\n", *date)
	quotes, err := tushareClient.DailyAll(ctx, tradeDate)
	if err != nil {
		log.Fatalf("fetch daily data: %v", err)
	}
	fmt.Printf("Got %d records\n", len(quotes))

	stockRepo := db.NewStockRepository()
	allStocks, err := stockRepo.GetActiveStocks(ctx)
	if err != nil {
		log.Fatalf("get active stocks: %v", err)
	}
	stockMap := make(map[string]*dal_model.StockBasicInfo, len(allStocks))
	for i := range allStocks {
		stockMap[allStocks[i].TsCode] = &allStocks[i]
	}

	boardRules := map[dal_model.BoardCode]*dal_model.BoardRule{
		dal_model.BoardMain: {BoardCode: dal_model.BoardMain, LimitUpRatio: 0.10},
		dal_model.BoardGEM:  {BoardCode: dal_model.BoardGEM, LimitUpRatio: 0.20},
		dal_model.BoardSTAR: {BoardCode: dal_model.BoardSTAR, LimitUpRatio: 0.20},
		dal_model.BoardBSE:  {BoardCode: dal_model.BoardBSE, LimitUpRatio: 0.30},
	}

	type job struct {
		tsCode string
		name   string
		pct    float64
		vol    int64
		amount float64
	}

	var limitUpJobs, above5Jobs []job
	filteredCount := 0
	skippedCount := 0

	for _, quote := range quotes {
		if quote.PctChg < 4.7 {
			filteredCount++
			continue
		}

		//if quote.TsCode != "002309.SZ" {
		//	continue
		//}

		stock, ok := stockMap[quote.TsCode]
		if !ok {
			skippedCount++
			continue
		}
		board := limiter.DetectBoard(quote.TsCode[:6])
		rule := boardRules[board]
		input := dal_model.DetectInput{
			TsCode:       quote.TsCode,
			StockName:    stock.Name,
			CurrentPrice: quote.Price,
			PreClose:     quote.PreClose,
			ChangePct:    quote.PctChg,
			BoardCode:    board,
			IsST:         stock.IsST,
			PrevState:    dal_model.LimitStateNone,
		}
		output := limiter.DetectLimitUp(input, rule)
		if output.Skipped {
			skippedCount++
			continue
		}
		j := job{tsCode: quote.TsCode, name: stock.Name, pct: quote.PctChg, vol: quote.Vol, amount: quote.Amount}
		if output.IsLimitUp {
			limitUpJobs = append(limitUpJobs, j)
		} else if output.IsAbove5Pct {
			above5Jobs = append(above5Jobs, j)
		}
	}

	fmt.Printf("Limit-up: %d, above5: %d, filtered(<4.7%%): %d, skipped: %d\n",
		len(limitUpJobs), len(above5Jobs), filteredCount, skippedCount)

	quoteTime, _ := time.Parse("2006-01-02", *date)
	quoteTime = quoteTime.Add(12*time.Hour + 30*time.Minute)

	classifyBatch := func(jobs []job) poolData {
		data := poolData{total: len(jobs), topics: make(map[string]topicData)}
		if len(jobs) == 0 {
			return data
		}
		tsCodes := make([]string, len(jobs))
		for i, j := range jobs {
			tsCodes[i] = j.tsCode
		}
		results, err := classifySvc.ClassifyStockBatch(ctx, tsCodes, *date, quoteTime)

		if err != nil {
			log.Fatalf("batch classify error: %v", err)
		}
		jobMap := make(map[string]job, len(jobs))
		for _, j := range jobs {
			jobMap[j.tsCode] = j
		}
		for _, tsCode := range tsCodes {
			j := jobMap[tsCode]
			mappings := results[tsCode]
			if len(mappings) == 0 {
				data.unclassified = append(data.unclassified, stockInfo{
					tsCode: j.tsCode, name: j.name, pct: j.pct, vol: j.vol, amount: j.amount,
				})
				continue
			}
			for _, m := range mappings {
				topicKey := m.TopicName
				if _, ok := data.topics[topicKey]; !ok {
					data.topics[topicKey] = topicData{
						meta: attributionInfo{
							TopicID: m.TopicID, TopicName: m.TopicName,
							HitCount: m.HitCount, LastSeenDate: m.LastSeenDate,
							Source: m.Source, Score: m.Confidence,
						},
					}
				}
				td := data.topics[topicKey]
				td.stocks = append(td.stocks, stockInfo{
					tsCode: j.tsCode, name: j.name, pct: j.pct, vol: j.vol, amount: j.amount,
					attr: attributionInfo{
						TopicID: m.TopicID, TopicName: m.TopicName,
						HitCount: m.HitCount, LastSeenDate: m.LastSeenDate,
						Source: m.Source, Score: m.Confidence,
					},
				})
				data.topics[topicKey] = td
			}
		}
		return data
	}

	limitUpPool := classifyBatch(limitUpJobs)
	limitUpPool.name = "涨停池"
	above5Pool := classifyBatch(above5Jobs)
	above5Pool.name = "5%池"

	var unclassifiedPool poolData
	unclassifiedPool.name = "未分类"
	unclassifiedPool.total = len(limitUpPool.unclassified) + len(above5Pool.unclassified)
	unclassifiedPool.topics = make(map[string]topicData)
	for _, s := range limitUpPool.unclassified {
		unclassifiedPool.unclassified = append(unclassifiedPool.unclassified, s)
	}
	for _, s := range above5Pool.unclassified {
		unclassifiedPool.unclassified = append(unclassifiedPool.unclassified, s)
	}

	fmt.Printf("Batch querying topic_relations for all pool stocks...\n")
	allPoolCodes := make([]string, 0, len(limitUpJobs)+len(above5Jobs))
	for _, s := range limitUpPool.unclassified {
		allPoolCodes = append(allPoolCodes, s.tsCode)
	}
	for _, td := range limitUpPool.topics {
		for _, s := range td.stocks {
			allPoolCodes = append(allPoolCodes, s.tsCode)
		}
	}
	for _, s := range above5Pool.unclassified {
		allPoolCodes = append(allPoolCodes, s.tsCode)
	}
	for _, td := range above5Pool.topics {
		for _, s := range td.stocks {
			allPoolCodes = append(allPoolCodes, s.tsCode)
		}
	}

	strRepo := db.NewStockTopicRelationRepository()
	allRelations, err := strRepo.GetByTsCodeBatch(ctx, allPoolCodes)
	if err != nil {
		log.Fatalf("batch get topic relations: %v", err)
	}
	fmt.Printf("Got %d topic_relations for %d stocks\n", len(allRelations), len(allPoolCodes))

	mergeRels := func(stocks []stockInfo) {
		for i := range stocks {
			if rels, ok := allRelations[stocks[i].tsCode]; ok {
				for _, r := range rels {
					ls := ""
					if r.LastSeenDate != nil {
						ls = r.LastSeenDate.Format("2006-01-02")
					}
					conf := 0.0
					if r.Confidence != nil {
						conf = *r.Confidence
					}
					stocks[i].rels = append(stocks[i].rels, relationInfo{
						TopicID: r.TopicID, TopicName: r.TopicName,
						HitCount: r.HitCount, LastSeenDate: ls,
						Source: r.Source, Confidence: conf,
					})
				}
				sort.Slice(stocks[i].rels, func(a, b int) bool {
					return stocks[i].rels[a].LastSeenDate > stocks[i].rels[b].LastSeenDate
				})
			}
		}
	}

	mergeRels(limitUpPool.unclassified)
	for k := range limitUpPool.topics {
		mergeRels(limitUpPool.topics[k].stocks)
	}
	mergeRels(above5Pool.unclassified)
	for k := range above5Pool.topics {
		mergeRels(above5Pool.topics[k].stocks)
	}
	mergeRels(unclassifiedPool.unclassified)

	genHTML(&limitUpPool, &above5Pool, &unclassifiedPool, *date, *mode, *outPath)
	fmt.Printf("\nHTML written to: %s\n", *outPath)
}

func genHTML(limitUp, above5, unclassified *poolData, date, mode, outPath string) {
	topicCount := len(limitUp.topics) + len(above5.topics)

	pools := []struct {
		id    string
		label string
		data  *poolData
	}{
		{"lu", "涨停池", limitUp},
		{"a5", "5%池", above5},
		{"uc", "未分类", unclassified},
	}

	var script strings.Builder
	script.WriteString(`

  function relsHTML(rels) {
    if (!rels || rels.length===0) return '<div style="font-size:12px;color:#6b7280;padding:4px 0;">无关联记录</div>';
    var h = '<div class="rel-list">';
    rels.forEach(function(r){
      h += '<div class="rel-item">';
      h += '<span class="rel-name">'+esc(r.tn)+'</span>';
      h += '<span class="rel-hit">'+r.h+'次</span>';
      h += '<span class="rel-date">'+r.ls+'</span>';
      h += '</div>';
    });
    h += '</div>';
    return h;
  }

  const pools = [`)

	for i, p := range pools {
		if i > 0 {
			script.WriteString(",")
		}
		script.WriteString(fmt.Sprintf("{id:'%s',label:'%s',total:%d,topics:{", p.id, p.data.name, p.data.total))
		tNames := make([]string, 0, len(p.data.topics))
		for tn := range p.data.topics {
			tNames = append(tNames, tn)
		}
		sort.Slice(tNames, func(i, j int) bool {
			return len(p.data.topics[tNames[i]].stocks) > len(p.data.topics[tNames[j]].stocks)
		})
		for j, tn := range tNames {
			if j > 0 {
				script.WriteString(",")
			}
			td := p.data.topics[tn]
			script.WriteString(fmt.Sprintf("'%s':{m:{id:%d,n:'%s',h:%d,ls:'%s',src:'%s',s:%.4f},stocks:[",
				escapeJS(tn), td.meta.TopicID, escapeJS(td.meta.TopicName),
				td.meta.HitCount, td.meta.LastSeenDate, td.meta.Source, td.meta.Score))
			for k, s := range td.stocks {
				if k > 0 {
					script.WriteString(",")
				}
				script.WriteString(fmt.Sprintf("{n:'%s',c:'%s',p:%.2f,v:%d,a:%.2f,h:%d,ls:'%s',src:'%s',s:%.4f,rels:[",
					escapeJS(s.name), s.tsCode, s.pct, s.vol, s.amount,
					s.attr.HitCount, s.attr.LastSeenDate, s.attr.Source, s.attr.Score))
				for l, r := range s.rels {
					if l > 0 {
						script.WriteString(",")
					}
					script.WriteString(fmt.Sprintf("{tid:%d,tn:'%s',h:%d,ls:'%s',src:'%s',c:%.2f}",
						r.TopicID, escapeJS(r.TopicName), r.HitCount, r.LastSeenDate, r.Source, r.Confidence))
				}
				script.WriteString("]}")
			}
			script.WriteString("]}")
		}
		script.WriteString("},unclassified:[")
		for k, s := range p.data.unclassified {
			if k > 0 {
				script.WriteString(",")
			}
			script.WriteString(fmt.Sprintf("{n:'%s',c:'%s',p:%.2f,v:%d,a:%.2f,rels:[", escapeJS(s.name), s.tsCode, s.pct, s.vol, s.amount))
			for l, r := range s.rels {
				if l > 0 {
					script.WriteString(",")
				}
				script.WriteString(fmt.Sprintf("{tid:%d,tn:'%s',h:%d,ls:'%s',src:'%s',c:%.2f}",
					r.TopicID, escapeJS(r.TopicName), r.HitCount, r.LastSeenDate, r.Source, r.Confidence))
			}
			script.WriteString("]}")
		}
		script.WriteString("]}")
	}
	script.WriteString(`];

  function esc(s) { return s.replace(/'/g,"\\'").replace(/"/g,'\\"'); }
  function fmtAmount(v) {
    if (v >= 1e8) return (v/1e8).toFixed(2)+'亿';
    if (v >= 1e4) return (v/1e4).toFixed(2)+'万';
    return v.toFixed(0);
  }

  function renderPool(poolId) {
    const pool = pools.find(p=>p.id===poolId);
    const container = document.getElementById('pool-'+poolId);
    const searchInput = document.getElementById('search-'+poolId);
    const activeTopic = document.querySelector('#sec-'+poolId+' .topic-filter-btn.active')?.dataset.topic;

    let topicNames = Object.keys(pool.topics);
    let html = '';

    topicNames.forEach(tn=>{
      const td = pool.topics[tn];
      const visible = (!activeTopic || activeTopic==='__all__' || activeTopic===esc(tn));
      const searchMatch = !searchInput.value || tn.includes(searchInput.value)
        || td.stocks.some(s=>s.n.includes(searchInput.value)||s.c.includes(searchInput.value));
      if (!visible || !searchMatch) return;

      html += '<div class="topic-section" data-topic="'+esc(tn)+'">';
      html += '<div class="topic-header" onclick="this.parentElement.classList.toggle(\'collapsed\')">';
      html += '<div class="topic-name"><span class="collapse-icon">▼</span>'+esc(tn)+'<span class="count">'+td.stocks.length+'只</span></div>';
      html += '</div>';
      html += '<div class="topic-body">';

      html += '<div class="attr-meta">';
      html += '<div class="attr-item">topic_id: <span>'+td.m.id+'</span></div>';
      html += '<div class="attr-item">hit_count: <span>'+td.m.h+'</span></div>';
      html += '<div class="attr-item">last_seen: <span>'+td.m.ls+'</span></div>';
      html += '<div class="attr-item source">source: <span>'+td.m.src+'</span></div>';
      html += '<div class="attr-item">score: <span>'+td.m.s.toFixed(4)+'</span></div>';
      html += '</div>';

      html += '<div style="overflow-x:auto;"><table class="topic-table"><thead class="table-head"><tr>';
      html += '<th style="width:30px"></th><th>股票名称</th><th>代码</th><th>涨跌幅</th><th>成交量</th><th>成交额</th>';
      html += '</tr></thead><tbody>';
      td.stocks.forEach(s=>{
        const rowShow = !searchInput.value || s.n.includes(searchInput.value) || s.c.includes(searchInput.value);
        const detailId = 'detail-'+poolId+'-'+s.c;
        html += '<tr class="table-row'+(rowShow?'':' hidden')+'" data-search="'+esc(s.n)+' '+esc(s.c)+'">';
        html += '<td><button class="stock-toggle" id="'+detailId+'-btn" onclick="toggleStock(\''+detailId+'\',\''+detailId+'-btn\')">▶</button></td>';
        html += '<td class="stock-name">'+esc(s.n)+'</td>';
        html += '<td class="ts-code">'+s.c+'</td>';
        html += '<td class="pct '+(s.p>=9.9?'up':'up5')+'">'+s.p.toFixed(2)+'%</td>';
        html += '<td class="num">'+s.v.toLocaleString()+'</td>';
        html += '<td class="num">'+fmtAmount(s.a)+'</td>';
        html += '</tr>';
        html += '<tr class="stock-detail" id="'+detailId+'"><td colspan="6">'+relsHTML(s.rels)+'</td></tr>';
      });
      html += '</tbody></table></div></div></div>';
    });

    if (pool.unclassified.length > 0) {
      const ucVisible = !activeTopic || activeTopic==='__all__';
      const ucSearch = !searchInput.value || pool.unclassified.some(s=>s.n.includes(searchInput.value)||s.c.includes(searchInput.value));
      if (ucVisible && ucSearch) {
        html += '<div class="topic-section" data-topic="__uc__">';
        html += '<div class="topic-header" onclick="this.parentElement.classList.toggle(\'collapsed\')">';
        html += '<div class="topic-name"><span class="collapse-icon">▼</span>未分类<span class="count">'+pool.unclassified.length+'只</span></div>';
        html += '</div>';
        html += '<div class="topic-body">';
        html += '<div style="overflow-x:auto;"><table class="topic-table"><thead class="table-head"><tr>';
        html += '<th style="width:30px"></th><th>股票名称</th><th>代码</th><th>涨跌幅</th><th>成交量</th><th>成交额</th>';
        html += '</tr></thead><tbody>';
        pool.unclassified.forEach(s=>{
          const rowShow = !searchInput.value || s.n.includes(searchInput.value) || s.c.includes(searchInput.value);
          const detailId = 'detail-'+poolId+'-'+s.c;
          html += '<tr class="table-row'+(rowShow?'':' hidden')+'" data-search="'+esc(s.n)+' '+esc(s.c)+'">';
          html += '<td><button class="stock-toggle" id="'+detailId+'-btn" onclick="toggleStock(\''+detailId+'\',\''+detailId+'-btn\')">▶</button></td>';
          html += '<td class="stock-name">'+esc(s.n)+'</td>';
          html += '<td class="ts-code">'+s.c+'</td>';
          html += '<td class="pct up">'+s.p.toFixed(2)+'%</td>';
          html += '<td class="num">'+s.v.toLocaleString()+'</td>';
          html += '<td class="num">'+fmtAmount(s.a)+'</td>';
          html += '</tr>';
          html += '<tr class="stock-detail" id="'+detailId+'"><td colspan="6">'+relsHTML(s.rels)+'</td></tr>';
        });
        html += '</tbody></table></div></div></div>';
      }
    }

    if (html==='') {
      html='<div class="empty-state">暂无数据</div>';
    }
    container.innerHTML = html;
  }

  window.toggleStock = function(detailId, btnId) {
    const detail = document.getElementById(detailId);
    const btn = document.getElementById(btnId);
    if (!detail) return;
    const isOpen = detail.classList.contains('open');
    detail.classList.toggle('open');
    if (btn) {
      if (isOpen) {
        btn.textContent = '▶';
        btn.classList.remove('open');
      } else {
        btn.textContent = '▼';
        btn.classList.add('open');
      }
    }
  };

  function initPools() {
    const tabBar = document.getElementById('tabBar');
    const container = document.getElementById('poolContainer');
    tabBar.innerHTML='';
    container.innerHTML='';

    pools.forEach((p,i)=>{
      const tab = document.createElement('div');
      tab.className='tab'+(i===0?' active':'')+(p.id==='uc'?' uc':'');
      tab.dataset.pool=p.id;
      tab.innerHTML=p.label+'<span class="tab-count">'+p.total+'</span>';
      tab.onclick=()=>{
        document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
        tab.classList.add('active');
        document.querySelectorAll('.pool-section').forEach(s=>s.classList.add('hidden'));
        document.getElementById('sec-'+p.id).classList.remove('hidden');
      };
      tabBar.appendChild(tab);

      const sec = document.createElement('div');
      sec.id='sec-'+p.id;
      sec.className='pool-section'+(i>0?' hidden':'');

      const filterDiv = document.createElement('div');
      filterDiv.className='pool-header';
      filterDiv.innerHTML='<div class="pool-title">'+p.label+' <span class="badge">'+p.total+'只</span></div>';
      filterDiv.innerHTML+='<div class="controls">';
      filterDiv.innerHTML+='<input class="search-input" id="search-'+p.id+'" placeholder="搜索股票..." oninput="renderPool(\''+p.id+'\')"> ';

      const allBtn = document.createElement('button');
      allBtn.className='topic-filter-btn all active';
      allBtn.dataset.topic='__all__';
      allBtn.textContent='全部';
      allBtn.onclick=()=>{
        document.querySelectorAll('#sec-'+p.id+' .topic-filter-btn').forEach(b=>b.classList.remove('active'));
        allBtn.classList.add('active');
        renderPool(p.id);
      };
      filterDiv.appendChild(allBtn);

      Object.keys(p.topics).forEach(tn=>{
        const btn = document.createElement('button');
        btn.className='topic-filter-btn';
        btn.dataset.topic=esc(tn);
        btn.textContent=esc(tn);
        btn.onclick=()=>{
          document.querySelectorAll('#sec-'+p.id+' .topic-filter-btn').forEach(b=>b.classList.remove('active'));
          btn.classList.add('active');
          renderPool(p.id);
        };
        filterDiv.appendChild(btn);
      });

      if (p.unclassified.length > 0) {
        const ucBtn = document.createElement('button');
        ucBtn.className='topic-filter-btn';
        ucBtn.dataset.topic='__uc__';
        ucBtn.textContent='未分类('+p.unclassified.length+')';
        ucBtn.onclick=()=>{
          document.querySelectorAll('#sec-'+p.id+' .topic-filter-btn').forEach(b=>b.classList.remove('active'));
          ucBtn.classList.add('active');
          renderPool(p.id);
        };
        filterDiv.appendChild(ucBtn);
      }

      sec.appendChild(filterDiv);

      const body = document.createElement('div');
      body.id='pool-'+p.id;
      sec.appendChild(body);
      container.appendChild(sec);

      renderPool(p.id);
    });
  }

  document.addEventListener('DOMContentLoaded', initPools);
`)

	scriptStr := script.String()

	data := map[string]any{
		"Date":              date,
		"Mode":              mode,
		"GenTime":           time.Now().Format("2006-01-02 15:04:05"),
		"LimitUpTotal":      limitUp.total,
		"Above5Total":       above5.total,
		"TopicCount":        topicCount,
		"UnclassifiedTotal": unclassified.total,
		"Script":            scriptStr,
	}

	html := htmlTemplate
	for k, v := range data {
		placeholder := "{{." + k + "}}"
		var valStr string
		switch vi := v.(type) {
		case int:
			valStr = fmt.Sprintf("%d", vi)
		default:
			valStr = vi.(string)
		}
		html = strings.ReplaceAll(html, placeholder, valStr)
	}

	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		log.Fatalf("write html: %v", err)
	}
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
