@echo off
chcp 65001 >nul

:: scripts/generate-proto.bat
:: 设置路径
set PROTO_PATH=../api/proto
set OUT_PATH=.

echo 🔄 Generating Protocol Buffer files...

:: 创建输出目录
if not exist "../api/proto/user" mkdir "../api/proto/user"
if not exist "../api/proto/trading" mkdir "../api/proto/trading"

:: 生成用户服务代码
protoc ^
  --proto_path=%PROTO_PATH% ^
  --proto_path=../third_party ^
  --go_out=%OUT_PATH% ^
  --go_opt=paths=source_relative ^
  --go-grpc_out=%OUT_PATH% ^
  --go-grpc_opt=paths=source_relative ^
  --grpc-gateway_out=%OUT_PATH% ^
  --grpc-gateway_opt=paths=source_relative ^
  --openapiv2_out=docs ^
  --openapiv2_opt=logtostderr=true ^
  user/user.proto

:: 生成交易服务代码
protoc ^
  --proto_path=%PROTO_PATH% ^
  --proto_path=../third_party ^
  --go_out=%OUT_PATH% ^
  --go_opt=paths=source_relative ^
  --go-grpc_out=%OUT_PATH% ^
  --go-grpc_opt=paths=source_relative ^
  --grpc-gateway_out=%OUT_PATH% ^
  --grpc-gateway_opt=paths=source_relative ^
  --openapiv2_out=docs ^
  --openapiv2_opt=logtostderr=true ^
  trading/trading.proto

echo ✅ Protocol Buffer files generated successfully!
pause