Write-Output "Cleaning previous build..."
Remove-Item better-dns.exe -ErrorAction SilentlyContinue
Write-Output "Building better-dns..."
go build better-dns.go
Write-Output "Starting better-dns..."
.\better-dns.exe
