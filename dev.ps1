Remove-Item better-dns.exe -ErrorAction SilentlyContinue
go build better-dns.go
.\better-dns.exe
