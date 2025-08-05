param(
    [string]$HostAddress = "localhost:8080",
    [string]$Ppm = "5.65",
    [string]$Token = ""
)

$process = Start-Process -FilePath ".\starfire.exe" -ArgumentList "-host", $HostAddress, "-ppm", $Ppm, "-token", $Token -WindowStyle Hidden -PassThru
Write-Host "client is running as daemon process , PID: $($process.Id)"
Write-Host "use 'taskkill /PID $($process.Id)' to stop the client"

# powershell -ExecutionPolicy Bypass -File start.ps1 -Host "1.94.239.51" -Ppm "5.65" -Token "12345678"