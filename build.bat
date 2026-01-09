@echo off
echo ========================================
echo    Market Notifier - EXE 빌드 (Go)
echo ========================================
echo.

echo [빌드] Windows EXE 파일 생성 중...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64

go build -ldflags="-s -w" -o MarketNotifier.exe main.go

if exist MarketNotifier.exe (
    echo.
    echo ========================================
    echo    빌드 완료!
    echo    MarketNotifier.exe 파일이 생성되었습니다.
    echo ========================================
) else (
    echo.
    echo [오류] 빌드에 실패했습니다.
    echo Go가 설치되어 있는지 확인하세요.
)

pause
