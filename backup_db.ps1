# Резервное копирование БД «Монтажник Связи» с ротацией.
# Использование:
#   powershell -File backup_db.ps1            — сделать бэкап
#   powershell -File backup_db.ps1 -Keep 30   — хранить 30 копий
# Восстановление:  pg_restore или psql -f из файла бэкапа.

param(
    [int]$Keep = 14
)

$ErrorActionPreference = "Stop"

$pgBin   = "C:\1.2 MS\postgres\pgsql\bin"
$backupDir = "C:\1.2 MS\backups"
$dbName  = "montazhnik"
$dbUser  = "montazhnik"
$dbPass  = if ($env:PGPASSWORD) { $env:PGPASSWORD } else { "montazhnik_pass" }

if (-not (Test-Path $backupDir)) {
    New-Item -ItemType Directory -Path $backupDir -Force | Out-Null
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$file  = Join-Path $backupDir "montazhnik-$stamp.dump"

Write-Host "Бэкап: $file"

$env:PGPASSWORD = $dbPass
& "$pgBin\pg_dump.exe" -U $dbUser -h localhost -d $dbName -F c -f $file -Z 9 2>&1 | Out-Host

if (-not (Test-Path $file)) {
    Write-Host "ОШИБКА: бэкап не создан" -ForegroundColor Red
    exit 1
}

$size = [math]::Round((Get-Item $file).Length / 1MB, 2)
Write-Host "Бэкап создан: $size МБ" -ForegroundColor Green

# Ротация: удаляем старые копии сверх $Keep.
$old = Get-ChildItem $backupDir -Filter "montazhnik-*.dump" |
    Sort-Object Name -Descending |
    Select-Object -Skip $Keep

foreach ($f in $old) {
    Write-Host "Удаляю старый бэкап: $($f.Name)"
    Remove-Item $f.FullName -Force
}

$count = (Get-ChildItem $backupDir -Filter "montazhnik-*.dump").Count
Write-Host "Хранится копий: $count"