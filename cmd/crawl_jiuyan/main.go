package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"stock/internal/external/jiuyan"
	"strings"
	"time"

	"gorm.io/gorm"
	"stock/config"
	"stock/dal/db"
	"stock/model/dal_model"
)

// mask 隐藏敏感字符串中间部分，只显示前4位和后4位
func mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// CurlParams 从浏览器复制的 curl 命令中提取 token、cookie、timestamp、date
type CurlParams struct {
	Token     string
	Cookie    string
	Timestamp string
	Date      string
}

// parseCurl 解析浏览器 DevTools 复制的 curl 命令，提取认证参数
// 用法：go run cmd/crawl_jiuyan/main.go -curl "粘贴完整curl命令"
func parseCurl(curlCmd string) (*CurlParams, error) {
	curlCmd = strings.TrimSpace(curlCmd)

	// 提取 Cookie: -b '...' 或 -b "..."
	cookieRegex := regexp.MustCompile(`-b\s+['"]([^'"]+)['"]`)
	cookieMatch := cookieRegex.FindStringSubmatch(curlCmd)
	if len(cookieMatch) < 2 {
		return nil, fmt.Errorf("未找到 Cookie，请确认 curl 命令包含 -b 参数")
	}

	// 提取 Token: -H 'token: xxx'
	tokenRegex := regexp.MustCompile(`-H\s+['"]token:\s*([^'"]+)['"]`)
	tokenMatch := tokenRegex.FindStringSubmatch(curlCmd)
	if len(tokenMatch) < 2 {
		return nil, fmt.Errorf("未找到 Token，请确认 curl 命令包含 -H 'token: ...'")
	}

	// 提取 Timestamp: -H 'timestamp: xxx'
	tsRegex := regexp.MustCompile(`-H\s+['"]timestamp:\s*([^'"]+)['"]`)
	tsMatch := tsRegex.FindStringSubmatch(curlCmd)
	if len(tsMatch) < 2 {
		return nil, fmt.Errorf("未找到 Timestamp，请确认 curl 命令包含 -H 'timestamp: ...'")
	}

	// 提取 Date: --data-raw '{"date":"YYYY-MM-DD",...}'
	dateRegex := regexp.MustCompile(`--data-raw\s+['"]\{[^}]*"date"\s*:\s*"(\d{4}-\d{2}-\d{2})"`)
	dateMatch := dateRegex.FindStringSubmatch(curlCmd)
	date := ""
	if len(dateMatch) >= 2 {
		date = dateMatch[1]
	}

	return &CurlParams{
		Token:     tokenMatch[1],
		Cookie:    cookieMatch[1],
		Timestamp: tsMatch[1],
		Date:      date,
	}, nil
}

func main() {
	startDate := flag.String("start", "2026-03-23", "起始日期 YYYY-MM-DD")
	endDate := flag.String("end", "", "结束日期 YYYY-MM-DD，默认为今天")
	curlCmd := flag.String("curl", "curl 'https://app.jiuyangongshe.com/jystock-app/api/v1/action/field' \\\n  -H 'Accept: application/json, text/plain, */*' \\\n  -H 'Accept-Language: zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6' \\\n  -H 'Connection: keep-alive' \\\n  -H 'Content-Type: application/json' \\\n  -b 'SESSION=ZjY3ZTZkNjgtNDAyMC00YmNmLTlkMGMtZWZjOGJmZGExMjVm; Hm_lvt_58aa18061df7855800f2a1b32d6da7f4=1773930659,1774175854,1774273568; Hm_lpvt_58aa18061df7855800f2a1b32d6da7f4=1774273568' \\\n  -H 'Origin: https://www.jiuyangongshe.com' \\\n  -H 'Referer: https://www.jiuyangongshe.com/' \\\n  -H 'Sec-Fetch-Dest: empty' \\\n  -H 'Sec-Fetch-Mode: cors' \\\n  -H 'Sec-Fetch-Site: same-site' \\\n  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0' \\\n  -H 'platform: 3' \\\n  -H 'sec-ch-ua: \"Chromium\";v=\"146\", \"Not-A.Brand\";v=\"24\", \"Microsoft Edge\";v=\"146\"' \\\n  -H 'sec-ch-ua-mobile: ?0' \\\n  -H 'sec-ch-ua-platform: \"Windows\"' \\\n  -H 'timestamp: 1774365145701' \\\n  -H 'token: 8b61415488d6cbebf3e7cd1954956e97' \\\n  --data-raw '{\"date\":\"2026-03-23\",\"pc\":1}'", "直接从浏览器复制完整的 curl 命令（包含 -b -H token -H timestamp 等），程序会自动提取 token/cookie/timestamp")
	token := flag.String("token", "", "Jiuyan Token（从 -curl 参数自动提取，也可手动指定）")
	cookie := flag.String("cookie", "", "Jiuyan Cookie（从 -curl 参数自动提取，也可手动指定）")
	timestamp := flag.String("timestamp", "", "Jiuyan Timestamp（从 -curl 参数自动提取，也可手动指定）")
	flag.Parse()

	// 如果传了 -curl，则从中提取参数
	if *curlCmd != "" {
		params, err := parseCurl(*curlCmd)
		if err != nil {
			log.Fatalf("解析 curl 命令失败: %v", err)
		}
		log.Printf("从 curl 中提取到: token=%s, cookie=%s, timestamp=%s, date=%s",
			mask(params.Token), mask(params.Cookie), params.Timestamp, params.Date)
		if *token == "" {
			*token = params.Token
		}
		if *cookie == "" {
			*cookie = params.Cookie
		}
		if *timestamp == "" {
			*timestamp = params.Timestamp
		}
		*startDate = params.Date
	}

	if *token == "" || *cookie == "" {
		log.Fatal("token 和 cookie 不能为空，请通过 -curl 参数提供完整 curl 命令，或手动指定 -token 和 -cookie")
	}

	if *token == "" || *cookie == "" {
		log.Fatal("token 和 cookie 参数必填，请从浏览器开发者工具获取")
	}

	if *endDate == "" {
		*endDate = time.Now().Format("2006-01-02")
	}

	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("invalid start date: %v", err)
	}
	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		log.Fatalf("invalid end date: %v", err)
	}

	ctx := context.Background()
	if err := config.Init("config/config.yaml"); err != nil {
		log.Fatalf("init config: %v", err)
	}

	db.Init()

	totalDays := 0
	successDays := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		weekday := d.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			continue
		}
		totalDays++
		dateStr := d.Format("2006-01-02")

		if err := crawlOneDate(ctx, db.DB, dateStr, *token, *cookie, *timestamp); err != nil {
			log.Printf("crawl %s failed: %v", dateStr, err)
			time.Sleep(2 * time.Second)
			continue
		}
		successDays++

		if !d.Equal(end) {
			time.Sleep(800 * time.Millisecond)
		}
	}

	log.Printf("range crawl done: %d/%d days succeeded", successDays, totalDays)
}

type jiuyanField struct {
	Name          string        `json:"name"`
	ActionFieldID string        `json:"action_field_id"`
	List          []jiuyanStock `json:"list"`
}

type jiuyanStock struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Article struct {
		ActionInfo struct {
			Expound string `json:"expound"`
		} `json:"action_info"`
	} `json:"article"`
}

type jiuyanRawResp struct {
	Data []jiuyanField `json:"data"`
}

func curlJiuyan(date, token, cookie string, timestamp string) ([]jiuyanField, error) {
	dataBody := fmt.Sprintf(`{"date":"%s","pc":1}`, date)

	args := []string{
		"-s", "-X", "POST",
		"-x", "http://127.0.0.1:7890",
		"https://app.jiuyangongshe.com/jystock-app/api/v1/action/field",
		"-H", "Accept: application/json, text/plain, */*",
		"-H", "Accept-Language: zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
		"-H", "Connection: keep-alive",
		"-H", "Content-Type: application/json",
		"-H", fmt.Sprintf("token: %s", token),
		"-H", fmt.Sprintf("timestamp: %s", timestamp),
		"-H", "platform: 3",
		"-H", "Origin: https://www.jiuyangongshe.com",
		"-H", "Referer: https://www.jiuyangongshe.com/",
		"-H", "Sec-Fetch-Dest: empty",
		"-H", "Sec-Fetch-Mode: cors",
		"-H", "Sec-Fetch-Site: same-site",
		"-H", "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0",
		"-H", "sec-ch-ua: \"Chromium\";v=\"146\", \"Not-A.Brand\";v=\"24\", \"Microsoft Edge\";v=\"146\"",
		"-H", "sec-ch-ua-mobile: ?0",
		"-H", "sec-ch-ua-platform: \"Windows\"",
		"-b", cookie,
		"--data-raw", dataBody,
	}

	cmd := exec.Command("curl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("curl failed: %w, stderr: %s", err, stderr.String())
	}

	var raw jiuyanRawResp
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("json decode: %w, body: %s", err, stdout.String())
	}

	if len(raw.Data) <= 1 {
		return []jiuyanField{}, nil
	}
	return raw.Data[1:], nil
}

func crawlOneDate(ctx context.Context, gormDB *gorm.DB, date, token, cookie string, timestamp string) error {
	log.Printf("fetching jiuyan data for %s", date)
	//fields, err := curlJiuyan(date, token, cookie, timestamp)
	fields, err := jiuyan.FetchFieldData(ctx, date)
	//fields, err := jiuyan.FetchFieldData(ctx, date)
	if err != nil {
		return fmt.Errorf("fetch field data: %w", err)
	}

	log.Printf("fetched %d topics", len(fields))

	// Load topic_dictionary into cache for normalization
	topicDictRepo := db.NewTopicDictionaryRepository()
	topicDictMap, err := topicDictRepo.GetAllMap(ctx)
	if err != nil {
		log.Printf("warn: load topic_dictionary failed: %v, using raw topic names", err)
		topicDictMap = make(map[string]dal_model.TopicDictionary)
	}

	var totalStocks int
	for _, field := range fields {
		totalStocks += len(field.List)

		// Use normalized name from topic_dictionary if available
		topicName := field.Name
		if dict, ok := topicDictMap[field.Name]; ok {
			topicName = dict.NormalizedName
		}

		for _, stock := range field.List {
			rawJSON, _ := json.Marshal(stock)
			err := gormDB.WithContext(ctx).Exec(`
				INSERT INTO jiuyan_raw_data (date, topic_name, action_field_id, stock_code, stock_name, expound, raw_json)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (date, topic_name, stock_code) DO UPDATE SET
					stock_name = EXCLUDED.stock_name,
					expound = EXCLUDED.expound,
					raw_json = EXCLUDED.raw_json
			`, date, topicName, field.ActionFieldID, stock.Code, stock.Name, stock.Article.ActionInfo.Expound, rawJSON).Error
			if err != nil {
				log.Printf("insert raw data failed: %v", err)
			}
		}

		parsedDate, _ := time.Parse("2006-01-02", date)
		err = gormDB.WithContext(ctx).Exec(`
			INSERT INTO topics (name, source, jiuyan_field_id, first_seen_date, last_seen_date, occurrence_count, updated_at)
			VALUES (?, 'jiuyan', ?, ?, ?, 1, NOW())
			ON CONFLICT (name) DO UPDATE SET
				jiuyan_field_id = COALESCE(EXCLUDED.jiuyan_field_id, topics.jiuyan_field_id),
				last_seen_date = GREATEST(topics.last_seen_date, EXCLUDED.last_seen_date),
				occurrence_count = topics.occurrence_count + 1,
				updated_at = NOW()
		`, topicName, field.ActionFieldID, parsedDate, parsedDate).Error
		if err != nil {
			log.Printf("upsert topic %s failed: %v", topicName, err)
		}
	}

	time.Sleep(1 * time.Second)

	log.Printf("wrote %d topics, %d stocks to jiuyan_raw_data", len(fields), totalStocks)
	return nil
}
