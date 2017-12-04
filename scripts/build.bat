if not exist "build" mkdir build
cd ../

set GOOS=windows
set GOARCH=386
go build -o scripts/build/golish-i386.exe
rice append --exec scripts/build/golish-i386.exe

set GOOS=windows
set GOARCH=amd64
go build -o scripts/build/golish-amd64.exe
rice append --exec scripts/build/golish-amd64.exe

set GOOS=linux
set GOARCH=386
go build -o scripts/build/golish-linux-i386
rice append --exec scripts/build/golish-linux-i386

set GOOS=linux
set GOARCH=amd64
go build -o scripts/build/golish-linux-amd64
rice append --exec scripts/build/golish-linux-amd64
