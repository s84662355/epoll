# hera_proxy 应用
## 生产环境
```shell
go build   -o bin/heraProxy.exe  
```
 

## 启动命令 
```shell
./heraProxy.exe  --config_path=conf
```


## 配置文件 conf

```shell
{
	"TcpListenerAddress":[":12423"],
	"LogDir":"./log"            
}
 
``` 

## 代码格式化
```shell
 gofumpt -l -w .
```

 
