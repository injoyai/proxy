name="special"

GOOS=linux GOARCH=amd64 go build -v -ldflags="-w -s" -o ./$name
echo "Linux编译完成..."
echo "开始压缩..."
upx -9 -k "./$name"
if [ -f "./$name.~" ]; then
  rm "./$name.~"
fi
if [ -f "./$name.000" ]; then
  rm "./$name.000"
fi



sleep 8
