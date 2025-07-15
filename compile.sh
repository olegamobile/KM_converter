GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc fyne package -os windows -icon fareva.png
upx --best --lzma KM_converter.exe