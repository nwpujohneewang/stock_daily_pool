package classify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"

	"stock/internal/pkg/logger"
)

const (
	sectorCacheTTL  = 600 * time.Second
	sectorPageSize  = 100
	sectorPageCount = 5
)

type SectorFetcher struct {
	httpClient *http.Client
	mu         sync.RWMutex
	cached     map[string]float64
	cacheTime  time.Time
}

func NewSectorFetcher() *SectorFetcher {
	return &SectorFetcher{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type eastMoneyConceptResp struct {
	Data struct {
		Diff []eastMoneyConceptItem `json:"diff"`
	} `json:"data"`
}

type eastMoneyConceptItem struct {
	Name      string  `json:"f14"`
	ChangePct float64 `json:"f3"`
}

// FetchSectorChanges returns concept name → pct_chg map using East Money real-time API.
// Results are cached for 60 seconds. The date parameter is accepted for interface
// compatibility but ignored (data is always real-time).
func (f *SectorFetcher) FetchSectorChanges(_ context.Context, _ string) (map[string]float64, error) {
	f.mu.RLock()
	if time.Since(f.cacheTime) < sectorCacheTTL && f.cached != nil {
		result := f.cached
		f.mu.RUnlock()
		return result, nil
	}
	f.mu.RUnlock()

	result, err := f.fetchFromEastMoney()
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	if time.Since(f.cacheTime) < sectorCacheTTL && f.cached != nil {
		cached := f.cached
		f.mu.Unlock()
		return cached, nil
	}
	f.cached = result
	f.cacheTime = time.Now()
	f.mu.Unlock()

	logger.Info("fetch sector changes", zap.Int("count", len(result)))
	return result, nil
}

func (f *SectorFetcher) fetchFromEastMoney() (map[string]float64, error) {
	out := make(map[string]float64, sectorPageSize*sectorPageCount)

	for page := 1; page <= sectorPageCount; page++ {
		items, err := f.fetchPage(page)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Name != "" {
				out[item.Name] = item.ChangePct
			}
		}
		if len(items) < sectorPageSize {
			break
		}
	}
	return out, nil
}

func (f *SectorFetcher) fetchPage(page int) ([]eastMoneyConceptItem, error) {
	params := url.Values{}
	params.Set("pn", fmt.Sprintf("%d", page))
	params.Set("pz", fmt.Sprintf("%d", sectorPageSize))
	params.Set("po", "1")
	params.Set("np", "1")
	params.Set("ut", "bd1d9ddb04089700cf9c27f6f7426281")
	params.Set("fltt", "2")
	params.Set("invt", "2")
	params.Set("fid", "f3")
	params.Set("fs", "m:90 t:3 f:!50")
	params.Set("fields", "f14,f3")

	reqURL := "https://push2.eastmoney.com/api/qt/clist/get?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")
	req.Header.Set("Accept", "application/json,text/plain,*/*")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("page %d http get: %w", page, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("page %d read body: %w", page, err)
	}

	if resp.StatusCode != http.StatusOK {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("page %d http %d: %s", page, resp.StatusCode, preview)
	}

	var result eastMoneyConceptResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("page %d unmarshal: %w", page, err)
	}
	return result.Data.Diff, nil
}
