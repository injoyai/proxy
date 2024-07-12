name="proxy"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/linux/$name ./cmd/main.go
echo "Linux编译完成..."
echo "开始压缩..."
upx -9 -k "./bin/linux/$name"
if [ -f "./bin/linux/$name.~" ]; then
  rm "./bin/linux/$name.~"
fi
if [ -f "./bin/linux/$name.000" ]; then
  rm "./bin/linux/$name.000"
fi

GOOS=linux GOARCH=arm GOARM=7 go build -v -ldflags="-w -s" -o ./bin/ubuntu/$name ./cmd/main.go
echo "Ubuntu编译完成..."
echo "开始压缩..."
upx -9 -k "./bin/ubuntu/$name"
if [ -f "./bin/ubuntu/$name.~" ]; then
  rm "./bin/ubuntu/$name.~"
fi
if [ -f "./bin/ubuntu/$name.000" ]; then
  rm "./bin/ubuntu/$name.000"
fi

GOOS=windows GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/windows/$name.exe ./cmd/main.go
echo "Windows编译完成..."
echo "开始压缩..."
upx -9 -k "./bin/windows/$name.exe"
if [ -f "./bin/windows/$name.ex~" ]; then
  rm "./bin/windows/$name.ex~"
fi
if [ -f "./bin/windows/$name.000" ]; then
  rm "./bin/windows/$name.000"
fi

sleep 8
