@echo off

go get github.com/cratonica/2goarray

echo // +build windows > on_windows.go
echo // +build windows > off_windows.go
echo // +build windows > unknown_windows.go

echo // +build !windows > on_other.go
echo // +build !windows > off_other.go
echo // +build !windows > unknown_other.go

2goarray On icon < on.ico >> on_windows.go
2goarray On icon < on.png >> on_other.go

2goarray Off icon < off.ico >> off_windows.go
2goarray Off icon < off.png >> off_other.go

2goarray Unknown icon < unknown.ico >> unknown_windows.go
2goarray Unknown icon < unknown.png >> unknown_other.go
