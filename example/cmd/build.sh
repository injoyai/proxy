name="proxy"
path="./bin"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o $path/amd64/$name ./main.go
echo "Linux编译完成..."
echo "开始压缩..."
upx -9 -k "$path/amd64/$name"
if [ -f "$path/amd64/$name.~" ]; then
  rm "$path/amd64/$name.~"
fi
if [ -f "$path/amd64/$name.000" ]; then
  rm "$path/amd64/$name.000"
fi

GOOS=linux GOARCH=arm GOARM=7 go build -v -ldflags="-w -s" -o $path/arm/$name ./main.go
echo "Ubuntu编译完成..."
echo "开始压缩..."
upx -9 -k "$path/arm/$name"
if [ -f "$path/arm/$name.~" ]; then
  rm "$path/arm/$name.~"
fi
if [ -f "$path/arm/$name.000" ]; then
  rm "$path/arm/$name.000"
fi

GOOS=windows GOARCH=amd64 go build -v -ldflags="-w -s" -o $path/windows/$name.exe ./main.go
echo "Windows编译完成..."
echo "开始压缩..."
upx -9 -k "$path/windows/$name.exe"
if [ -f "$path/windows/$name.ex~" ]; then
  rm "$path/windows/$name.ex~"
fi
if [ -f "$path/windows/$name.000" ]; then
  rm "$path/windows/$name.000"
fi

sleep 8
