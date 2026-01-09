# PowerShell 스크립트 - 백그라운드에서 Market Notifier 실행

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Market Notifier - 백그라운드 실행" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$exePath = "$PSScriptRoot\MarketNotifier.exe"

if (Test-Path $exePath) {
    Write-Host "[실행] MarketNotifier.exe를 백그라운드에서 시작합니다..." -ForegroundColor Green
    Start-Process -FilePath $exePath -WindowStyle Hidden
    Write-Host "[완료] 프로그램이 백그라운드에서 실행 중입니다." -ForegroundColor Green
    Write-Host ""
    Write-Host "종료하려면: Get-Process MarketNotifier | Stop-Process" -ForegroundColor Yellow
} else {
    Write-Host "[오류] $exePath 파일이 없습니다." -ForegroundColor Red
    Write-Host "먼저 build.bat를 실행하여 EXE 파일을 생성하세요." -ForegroundColor Yellow
}

Write-Host ""
Read-Host "아무 키나 누르면 종료됩니다"
