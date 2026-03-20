package jiuyan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type FieldData struct {
	Name          string      `json:"name"`
	ActionFieldID string      `json:"action_field_id"`
	List          []StockItem `json:"list"`
}

type jiuyanResponse struct {
	Data []FieldData `json:"data"`
}

type StockItem struct {
	Code    string      `json:"code"`
	Name    string      `json:"name"`
	Article ArticleInfo `json:"article"`
}

type ArticleInfo struct {
	ActionInfo ActionDetail `json:"action_info"`
}

type ActionDetail struct {
	Expound string `json:"expound"`
}

func FetchFieldData(ctx context.Context, date string) ([]FieldData, error) {
	url := "https://app.jiuyangongshe.com/jystock-app/api/v1/action/field"

	reqBody := map[string]interface{}{"date": date, "pc": 1}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Cookie", "SESSION=ZjY3ZTZkNjgtNDAyMC00YmNmLTlkMGMtZWZjOGJmZGExMjVm; Hm_lvt_58aa18061df7855800f2a1b32d6da7f4=1773930659; Hm_lpvt_58aa18061df7855800f2a1b32d6da7f4=1774008841")
	req.Header.Set("Token", "17ea9cca36ab5445eda0ec3eda0773f4")
	req.Header.Set("Timestamp", "1774008851872")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Platform", "3")
	req.Header.Set("Origin", "https://www.jiuyangongshe.com")
	req.Header.Set("Referer", "https://www.jiuyangongshe.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Microsoft Edge";v="146"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")

	httpClient := &http.Client{Timeout: 30 * time.Second}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	fmt.Println(string(body))

	var raw jiuyanResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(raw.Data) <= 1 {
		return []FieldData{}, nil
	}

	return raw.Data[1:], nil
}
