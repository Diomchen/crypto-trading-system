# gen proto 相关执行说明

首先需要下载相关的google api 包

```bash
mkdir -p third_party/google/api
curl -o third_party/google/api/annotations.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto
curl -o third_party/google/api/http.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto
```


windows 执行 gen_proto.bat 脚本
```bash
cd scripts
.\gen_proto.bat
```

//TODO: linux 待补充