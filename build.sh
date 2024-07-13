name="proxy"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./bin/amd64/$name ./cmd/main.go
echo "Linux编译完成..."
echo "开始压缩..."
upx -9 -k "./bin/amd64/$name"
if [ -f "./bin/amd64/$name.~" ]; then
  rm "./bin/amd64/$name.~"
fi
if [ -f "./bin/amd64/$name.000" ]; then
  rm "./bin/amd64/$name.000"
fi

GOOS=linux GOARCH=arm GOARM=7 go build -v -ldflags="-w -s" -o ./bin/arm/$name ./cmd/main.go
echo "Ubuntu编译完成..."
echo "开始压缩..."
upx -9 -k "./bin/arm/$name"
if [ -f "./bin/arm/$name.~" ]; then
  rm "./bin/arm/$name.~"
fi
if [ -f "./bin/arm/$name.000" ]; then
  rm "./bin/arm/$name.000"
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
