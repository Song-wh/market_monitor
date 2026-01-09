# PowerShell 스크립트 - Market Notifier 종료

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "   Market Notifier - 프로세스 종료" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$process = Get-Process -Name "MarketNotifier" -ErrorAction SilentlyContinue

if ($process) {
    Write-Host "[종료] MarketNotifier 프로세스를 종료합니다..." -ForegroundColor Yellow
    $process | Stop-Process -Force
    Write-Host "[완료] 프로그램이 종료되었습니다." -ForegroundColor Green
} else {
    Write-Host "[알림] 실행 중인 MarketNotifier 프로세스가 없습니다." -ForegroundColor Gray
}

Write-Host ""
Read-Host "아무 키나 누르면 종료됩니다"
