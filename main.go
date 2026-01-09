package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// ============================================
// 설정 - 여기에 정보를 입력하세요
// ============================================
const (
	// 텔레그램 설정
	TelegramBotToken = "8422283619:AAHtEQyjJR2t0qkn6HlA1cDAWhWIQdo1RQ8"
	TelegramChatID   = "-5219582928"

	// 한국수출입은행 API 키 (https://www.koreaexim.go.kr/ir/HPHKIR020M01 에서 발급)
	KoreaEximAPIKey = "YOUR_KOREAEXIM_API_KEY"

	// 한국석유공사 Opinet API 키
	OpinetAPIKey = "F260109036"

	// 알림 시간 설정
	MarketOpenTime  = "09:00"
	MarketCloseTime = "15:30"
	SkipWeekends    = true
)

// ============================================
// 데이터 구조체
// ============================================

// 한국수출입은행 환율 응답
type KoreaEximRate struct {
	Result      int    `json:"result"`       // 조회 결과 (1: 성공)
	CurUnit     string `json:"cur_unit"`     // 통화 코드
	CurNm       string `json:"cur_nm"`       // 통화 이름
	Ttb         string `json:"ttb"`          // 전신환 매입률
	Tts         string `json:"tts"`          // 전신환 매도율
	DealBasR    string `json:"deal_bas_r"`   // 매매기준율
	BkprBuyR    string `json:"bkpr"`         // 장부가격(매입)
	YyEfeeR     string `json:"yy_efee_r"`    // 연환가료율
	TenDdEfeeR  string `json:"ten_dd_efee_r"` // 10일환가료율
	KftcBkpr    string `json:"kftc_bkpr"`    // 서울외국환중개 매매기준율
	KftcDealBasR string `json:"kftc_deal_bas_r"` // 서울외국환중개 장부가격
}

// 한국석유공사 Opinet 유가 응답
type OpinetOilPrice struct {
	Result struct {
		Oil []struct {
			ProdCd string `json:"PRODCD"` // 제품코드
			Price  string `json:"PRICE"`  // 전국 평균가격
			Diff   string `json:"DIFF"`   // 전일대비
		} `json:"OIL"`
	} `json:"RESULT"`
}

// Yahoo Finance 응답 (금 시세용)
type YahooFinanceResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

// Fallback: ExchangeRate-API 응답
type ExchangeRateAPIResponse struct {
	Rates map[string]float64 `json:"rates"`
}

// ============================================
// HTTP 클라이언트
// ============================================

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

// ============================================
// 한국수출입은행 환율 조회 (공식 API)
// ============================================

func getKoreaEximRates() (map[string]float64, error) {
	// 오늘 날짜 (YYYYMMDD)
	today := time.Now().Format("20060102")
	
	apiURL := fmt.Sprintf(
		"https://www.koreaexim.go.kr/site/program/financial/exchangeJSON?authkey=%s&searchdate=%s&data=AP01",
		KoreaEximAPIKey, today,
	)

	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("API 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %v", err)
	}

	var rates []KoreaEximRate
	if err := json.Unmarshal(body, &rates); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	result := make(map[string]float64)
	for _, rate := range rates {
		// 쉼표 제거 후 숫자 변환
		priceStr := strings.ReplaceAll(rate.DealBasR, ",", "")
		var price float64
		fmt.Sscanf(priceStr, "%f", &price)

		switch rate.CurUnit {
		case "USD":
			result["USD"] = price
		case "EUR":
			result["EUR"] = price
		case "JPY(100)":
			result["JPY100"] = price
		case "CNH":
			result["CNY"] = price
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("환율 데이터 없음 (API 키 확인 필요)")
	}

	return result, nil
}

// Fallback: ExchangeRate-API (API 키 없을 때)
func getFallbackExchangeRates() (map[string]float64, error) {
	result := make(map[string]float64)

	currencies := []string{"USD", "EUR", "JPY"}
	for _, cur := range currencies {
		url := fmt.Sprintf("https://api.exchangerate-api.com/v4/latest/%s", cur)
		resp, err := httpClient.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var data ExchangeRateAPIResponse
		json.Unmarshal(body, &data)

		if cur == "JPY" {
			result["JPY100"] = data.Rates["KRW"] * 100
		} else {
			result[cur] = data.Rates["KRW"]
		}
	}

	return result, nil
}

// 환율 조회 (메인 함수)
func getExchangeRates() (map[string]float64, string, error) {
	// 한국수출입은행 API 시도
	if KoreaEximAPIKey != "YOUR_KOREAEXIM_API_KEY" {
		rates, err := getKoreaEximRates()
		if err == nil {
			return rates, "한국수출입은행", nil
		}
		fmt.Printf("[경고] 한국수출입은행 API 실패: %v, Fallback 사용\n", err)
	}

	// Fallback
	rates, err := getFallbackExchangeRates()
	return rates, "ExchangeRate-API", err
}

// ============================================
// 한국석유공사 Opinet 유가 조회 (공식 API)
// ============================================

func getOpinetOilPrices() (map[string]string, error) {
	apiURL := fmt.Sprintf(
		"https://www.opinet.co.kr/api/avgAllPrice.do?out=json&code=%s",
		OpinetAPIKey,
	)

	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("API 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %v", err)
	}

	var data OpinetOilPrice
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	result := make(map[string]string)
	for _, oil := range data.Result.Oil {
		switch oil.ProdCd {
		case "B027": // 휘발유
			result["휘발유"] = oil.Price
			result["휘발유_diff"] = oil.Diff
		case "D047": // 경유
			result["경유"] = oil.Price
			result["경유_diff"] = oil.Diff
		case "C004": // 등유
			result["등유"] = oil.Price
			result["등유_diff"] = oil.Diff
		case "K015": // LPG
			result["LPG"] = oil.Price
			result["LPG_diff"] = oil.Diff
		}
	}

	return result, nil
}

// ============================================
// 국제 유가 조회 (Yahoo Finance - WTI, Brent)
// ============================================

func getInternationalOilPrices() (wti, brent float64, err error) {
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}

	// WTI
	wti, _ = getYahooPrice("CL=F", headers)
	// Brent
	brent, _ = getYahooPrice("BZ=F", headers)

	return wti, brent, nil
}

func getYahooPrice(symbol string, headers map[string]string) (float64, error) {
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", symbol)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data YahooFinanceResponse
	json.Unmarshal(body, &data)

	if len(data.Chart.Result) == 0 {
		return 0, fmt.Errorf("no data")
	}

	return data.Chart.Result[0].Meta.RegularMarketPrice, nil
}

// 금 시세 조회
func getGoldPrice() (float64, error) {
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}
	return getYahooPrice("GC=F", headers)
}

// ============================================
// 텔레그램 메시지 전송
// ============================================

func sendTelegramMessage(message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", TelegramBotToken)

	data := url.Values{}
	data.Set("chat_id", TelegramChatID)
	data.Set("text", message)
	data.Set("parse_mode", "HTML")

	resp, err := httpClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}

	return nil
}

// ============================================
// 시장 메시지 생성
// ============================================

func createMarketMessage(eventType string) string {
	now := time.Now()
	dateStr := now.Format("2006년 01월 02일 15:04")

	var header, emoji string
	switch eventType {
	case "open":
		header = "🔔 <b>장시작 알림</b> 🔔"
		emoji = "🌅"
	case "close":
		header = "🔔 <b>장마감 알림</b> 🔔"
		emoji = "🌆"
	default: // start
		header = "🚀 <b>시장 알리미 시작</b> 🚀"
		emoji = "📊"
	}

	// 데이터 조회
	exchangeRates, exchangeSource, _ := getExchangeRates()
	wti, brent, _ := getInternationalOilPrices()
	gold, _ := getGoldPrice()
	domesticOil, opinetErr := getOpinetOilPrices()

	// 메시지 구성
	msg := fmt.Sprintf(`%s
📅 %s

━━━━━━━━━━━━━━━━━━━━
💱 <b>환율 정보</b> <i>(%s)</i>
━━━━━━━━━━━━━━━━━━━━
`, header, dateStr, exchangeSource)

	if len(exchangeRates) > 0 {
		if usd, ok := exchangeRates["USD"]; ok {
			msg += fmt.Sprintf("🇺🇸 USD/KRW: %.2f원\n", usd)
		}
		if eur, ok := exchangeRates["EUR"]; ok {
			msg += fmt.Sprintf("🇪🇺 EUR/KRW: %.2f원\n", eur)
		}
		if jpy, ok := exchangeRates["JPY100"]; ok {
			msg += fmt.Sprintf("🇯🇵 JPY(100)/KRW: %.2f원\n", jpy)
		}
		if cny, ok := exchangeRates["CNY"]; ok {
			msg += fmt.Sprintf("🇨🇳 CNY/KRW: %.2f원\n", cny)
		}
	} else {
		msg += "❌ 환율 정보를 가져올 수 없습니다.\n"
	}

	msg += `
━━━━━━━━━━━━━━━━━━━━
🛢️ <b>국제 유가</b>
━━━━━━━━━━━━━━━━━━━━
`
	if wti > 0 {
		msg += fmt.Sprintf("🇺🇸 WTI: $%.2f\n", wti)
	}
	if brent > 0 {
		msg += fmt.Sprintf("🇬🇧 Brent: $%.2f\n", brent)
	}
	if wti == 0 && brent == 0 {
		msg += "❌ 국제 유가 조회 실패\n"
	}

	// 국내 유가 (Opinet)
	if opinetErr == nil && len(domesticOil) > 0 {
		msg += `
━━━━━━━━━━━━━━━━━━━━
⛽ <b>국내 유가</b> <i>(전국 평균)</i>
━━━━━━━━━━━━━━━━━━━━
`
		if price, ok := domesticOil["휘발유"]; ok {
			diff := domesticOil["휘발유_diff"]
			diffIcon := getDiffIcon(diff)
			msg += fmt.Sprintf("⛽ 휘발유: %s원 %s\n", price, diffIcon)
		}
		if price, ok := domesticOil["경유"]; ok {
			diff := domesticOil["경유_diff"]
			diffIcon := getDiffIcon(diff)
			msg += fmt.Sprintf("🚛 경유: %s원 %s\n", price, diffIcon)
		}
	}

	msg += `
━━━━━━━━━━━━━━━━━━━━
🥇 <b>금 시세</b>
━━━━━━━━━━━━━━━━━━━━
`
	if gold > 0 {
		msg += fmt.Sprintf("💰 Gold: $%.2f/oz\n", gold)
	} else {
		msg += "❌ 금 시세 조회 실패\n"
	}

	msg += fmt.Sprintf("\n%s 좋은 투자 되세요! %s", emoji, emoji)

	return msg
}

func getDiffIcon(diff string) string {
	if strings.HasPrefix(diff, "-") {
		return "📉"
	} else if diff != "0" && diff != "" {
		return "📈"
	}
	return "➖"
}

// ============================================
// 스케줄러
// ============================================

func isWeekend() bool {
	weekday := time.Now().Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func parseTime(timeStr string) (hour, minute int) {
	fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	return
}

func notifyMarketOpen() {
	if SkipWeekends && isWeekend() {
		fmt.Println("[알림] 주말이므로 장시작 알림을 건너뜁니다.")
		return
	}

	fmt.Println("[알림] 장시작 알림 전송 중...")
	message := createMarketMessage("open")
	if err := sendTelegramMessage(message); err != nil {
		fmt.Printf("[오류] 메시지 전송 실패: %v\n", err)
	} else {
		fmt.Println("[성공] 장시작 알림 전송 완료!")
	}
}

func notifyMarketClose() {
	if SkipWeekends && isWeekend() {
		fmt.Println("[알림] 주말이므로 장마감 알림을 건너뜁니다.")
		return
	}

	fmt.Println("[알림] 장마감 알림 전송 중...")
	message := createMarketMessage("close")
	if err := sendTelegramMessage(message); err != nil {
		fmt.Printf("[오류] 메시지 전송 실패: %v\n", err)
	} else {
		fmt.Println("[성공] 장마감 알림 전송 완료!")
	}
}

func runScheduler() {
	openHour, openMin := parseTime(MarketOpenTime)
	closeHour, closeMin := parseTime(MarketCloseTime)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	var lastOpenDate, lastCloseDate string

	for range ticker.C {
		now := time.Now()
		today := now.Format("2006-01-02")
		hour, min := now.Hour(), now.Minute()

		// 장시작 알림
		if hour == openHour && min == openMin && lastOpenDate != today {
			lastOpenDate = today
			notifyMarketOpen()
		}

		// 장마감 알림
		if hour == closeHour && min == closeMin && lastCloseDate != today {
			lastCloseDate = today
			notifyMarketClose()
		}
	}
}

// ============================================
// 메인 함수
// ============================================

func main() {
	fmt.Println("==================================================")
	fmt.Println("📈 시장 알리미 (Market Notifier) - Go Version")
	fmt.Println("==================================================")
	fmt.Printf("장시작 알림 시간: %s\n", MarketOpenTime)
	fmt.Printf("장마감 알림 시간: %s\n", MarketCloseTime)
	fmt.Printf("주말 제외: %v\n", SkipWeekends)
	fmt.Println("==================================================")

	// API 키 상태 표시
	fmt.Println("\n📡 API 설정 상태:")
	if KoreaEximAPIKey != "YOUR_KOREAEXIM_API_KEY" {
		fmt.Println("  ✅ 한국수출입은행 API: 설정됨")
	} else {
		fmt.Println("  ⚠️  한국수출입은행 API: 미설정 (Fallback 사용)")
		fmt.Println("     → https://www.koreaexim.go.kr/ir/HPHKIR020M01 에서 발급")
	}
	if OpinetAPIKey != "YOUR_OPINET_API_KEY" {
		fmt.Println("  ✅ 한국석유공사 Opinet API: 설정됨")
	} else {
		fmt.Println("  ⚠️  한국석유공사 Opinet API: 미설정 (국내 유가 표시 안됨)")
		fmt.Println("     → https://www.opinet.co.kr/user/custapi/custApiInfo.do 에서 발급")
	}
	fmt.Println("==================================================")

	// 설정 확인
	if TelegramBotToken == "YOUR_BOT_TOKEN_HERE" {
		fmt.Println("\n⚠️  경고: 텔레그램 봇 토큰이 설정되지 않았습니다!")
		fmt.Println("main.go 파일에서 TelegramBotToken을 설정해주세요.")
		fmt.Println("\n아무 키나 누르면 종료됩니다...")
		fmt.Scanln()
		os.Exit(1)
	}

	if TelegramChatID == "YOUR_CHAT_ID_HERE" {
		fmt.Println("\n⚠️  경고: 텔레그램 채팅 ID가 설정되지 않았습니다!")
		fmt.Println("main.go 파일에서 TelegramChatID를 설정해주세요.")
		fmt.Println("\n아무 키나 누르면 종료됩니다...")
		fmt.Scanln()
		os.Exit(1)
	}

	// 시작 시 즉시 시장 정보 전송
	fmt.Println("\n[시작] 현재 시장 정보를 전송합니다...")
	startMsg := createMarketMessage("start")
	
	if err := sendTelegramMessage(startMsg); err != nil {
		fmt.Printf("[오류] 시작 알림 전송 실패: %v\n", err)
		fmt.Println("텔레그램 설정을 확인해주세요.")
		fmt.Println("\n아무 키나 누르면 종료됩니다...")
		fmt.Scanln()
		os.Exit(1)
	}
	fmt.Println("[성공] 시장 정보 전송 완료!")

	// 스케줄러 시작
	go runScheduler()

	fmt.Println("\n✅ 스케줄이 설정되었습니다. 대기 중...")
	fmt.Println("(Ctrl+C로 종료)\n")

	// 종료 시그널 대기
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n[종료] 프로그램을 종료합니다...")
	sendTelegramMessage("🔴 <b>시장 알리미가 종료되었습니다.</b>")
}
