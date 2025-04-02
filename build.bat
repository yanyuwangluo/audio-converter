set version=1.0.1
 
 
@REM 打包linux
set GOOS=linux
go env -w GOOS=linux
call go build -o audio-converter_%version%_linux_amd64 main.go
 
call set GOOS=windows
call go env -w GOOS=windows
call go build -o audio-converter_%version%_win.exe 