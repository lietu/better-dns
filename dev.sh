#!/usr/bin/env sh

echo "Cleaning previous build..."
rm -f better-dns
echo "Building better-dns..."
go build better-dns.go
echo "Starting better-dns..."
./better-dns
