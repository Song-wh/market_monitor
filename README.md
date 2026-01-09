# 📈 시장 알리미 (Market Notifier) - Go Version

한국 주식시장 장시작(09:00) 및 장마감(15:30) 시 환율과 유가 정보를 텔레그램으로 전송하는 프로그램입니다.

**Go로 작성되어 단일 EXE 파일로 실행됩니다. (의존성 없음)**

## 📋 기능

- 📊 **환율 정보**: USD/KRW, EUR/KRW, JPY/KRW
- 🛢️ **국제 유가**: WTI, Brent 원유 가격
- 🥇 **금 시세**: 국제 금 가격
- ⏰ **자동 알림**: 장시작(09:00), 장마감(15:30) 시 자동 전송
- 📅 **주말 제외**: 토/일요일은 알림 제외

## 🚀 설치 및 설정

### 1. Go 설치 (없는 경우)

[Go 다운로드](https://go.dev/dl/)에서 Windows용 설치파일 다운로드 후 설치

### 2. 텔레그램 봇 생성

1. 텔레그램에서 **@BotFather** 검색
2. `/newbot` 명령어 입력
3. 봇 이름과 username 설정
4. 발급받은 **토큰** 저장

### 3. 채팅 ID 확인

1. 생성한 봇에게 아무 메시지나 전송
2. 브라우저에서 접속:
   ```
   https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
   ```
3. `"chat": {"id": 123456789}` 에서 숫자가 채팅 ID

### 4. main.go 설정 수정

`main.go` 파일 상단의 설정값을 수정하세요:

```go
const (
    TelegramBotToken = "여기에_봇_토큰_입력"
    TelegramChatID   = "여기에_채팅_ID_입력"
    
    MarketOpenTime  = "09:00" // 장시작 시간
    MarketCloseTime = "15:30" // 장마감 시간
    SkipWeekends    = true    // 주말 알림 제외
)
```

## 💻 빌드 및 실행

### EXE 파일 생성

```powershell
# 프로젝트 폴더로 이동
cd c:\study\market-notifier

# 방법 1: build.bat 더블클릭

# 방법 2: 직접 빌드
go build -ldflags="-s -w" -o MarketNotifier.exe main.go
```

### 실행

```powershell
# 직접 실행
.\MarketNotifier.exe

# 백그라운드 실행
.\run_background.ps1

# PowerShell에서 직접 백그라운드 실행
Start-Process -FilePath ".\MarketNotifier.exe" -WindowStyle Hidden

# 종료
.\stop.ps1
```

## 📱 수신 메시지 예시

```
🔔 장시작 알림 🔔
📅 2026년 01월 09일 09:00

━━━━━━━━━━━━━━━━━━━━
💱 환율 정보
━━━━━━━━━━━━━━━━━━━━
🇺🇸 USD/KRW: 1350.50원
🇪🇺 EUR/KRW: 1480.30원
🇯🇵 JPY(100)/KRW: 920.15원

━━━━━━━━━━━━━━━━━━━━
🛢️ 국제 유가
━━━━━━━━━━━━━━━━━━━━
🇺🇸 WTI: $72.50
🇬🇧 Brent: $76.80

━━━━━━━━━━━━━━━━━━━━
🥇 금 시세
━━━━━━━━━━━━━━━━━━━━
💰 Gold: $2050.30/oz

🌅 좋은 투자 되세요! 🌅
```

## 🔧 트러블슈팅

### Go가 설치되지 않은 경우
```powershell
winget install GoLang.Go
```

### 빌드 오류 시
```powershell
go mod tidy
go build -o MarketNotifier.exe main.go
```

## 📝 라이선스

MIT License
