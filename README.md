# 介绍

类似于Grpc的使用流程，请搭配[hgen](https://github.com/gufeijun/hgen)编译器使用。

启动一个最简单的rpch服务的步骤如下：

**创建IDL**:

这里简单定义一个两数相加的服务：

```protobuf
//math.gfj
service Math{
    int32 Add(int32,int32)
}
```

编译器编译：`hgen -dir service -lang go math.gfj `。即会生成一个service/math.rpch.go文件，service是自动生成代码所在的包，其中包括了服务注册以及客户端调用服务的一些方法。

**server:**

```go
//自动生成的service/math.rpch.go：
//func RegisterMathService(impl MathService, svr *rpch.Server) 
//type MathService interface{
//	Add(int32, int32) (int32, error)
//}

type mathService struct{}

//实现service.MathService接口
func (*mathService) Add(a int32, b int32) (int32, error) {
	return a + b, nil
}

s := rpch.NewServer()
service.RegisterMathService(new(mathService),svr)	//此函数由编译器生成
svr.ListenAndServe("127.0.0.1:8080")
```

**client**：

```go
conn, _ := rpch.Dial(addr)
client := service.NewMathServiceClient(conn)	//此函数由编译器生成
result, _ := client.Add(2, 3)

//自动生成的service/math.rpch.go:
//func (c *MathServiceClient) Add(arg1 int32, arg2 int32) (res int32, err error) 
```

该仓库下的[examples](https://github.com/gufeijun/rpch-go/tree/master/examples)就是使用案例，目前在不停的增添中。欢迎您PR提交更多的案例。

# 协议设计

### 协议

传输层采用TCP，应用层协议单独设计，支持长连接。

客户端发起TCP连接后，需要完成握手过程：客户端发送4B的小端魔数(0x00686A6C)。如果服务端未正确接收到魔数，则断开TCP连接。

以IDL定义Add服务为例：

```go
service Math{
	uint32 Add(uint32,uint32)
}
```

客户端发送请求报文：

```
//第一行为请求行
Math Add 2 1\r\n //分别对应服务名 方法名 请求参数个数 请求的序号
TypeKind(2B) TypeNameLength(2B) DataLength(4B) TypeName Data //第一个参数
TypeKind(2B) TypeNameLength(2B) DataLength(4B) TypeName Data //第二个参数
```

请求序号方便用于开发异步请求客户端。使用TLV方式解决粘包问题，使用**小端方式**传输Type以及长度。TypeName为参数的类型(字符串方式)，Data为序列化后的数据。

服务端的响应报文：

```
Sequence(8B) TypeKind(2B) TypeNameLength(2B) DataLength(4B) TypeName Data
```

和客户端请求报文参数大同小异，不过多了8B的请求序号。

### 序列化

框架支持传输四种类型：

+ string

+ int32、uint32等能确定位长的Number数字类型。

+ 复合类型，即用message定义的类型。

+ Stream流类型。string类型不需要做序列化，数字类型采用小端方式即可，复合类型使用json传输，stream流类型使用
  http1.1引入的chunk编码实现。

string类型不需要做序列化，数字类型采用小端方式即可，复合类型使用json传输，stream流类型使用http1.1引入的chunk编码实现。

stream类型为本框架独创类型，能够让客户端宛如操纵本地文件一样操纵服务端的文件句柄。使用案例：

定义IDL服务：

```protobuf
service File{
	stream OpenFile(string)
}
```

服务端实现服务：

```go
//将本地的文件句柄直接返回
func (*fileServer) OpenFile(filepath string)(stream io.ReadWriter, OnFinish
func(), err error){
    file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
    if err !=nil{
    	return
    }
    return file, func(){
    	file.Close()
    },nil
}
```

客户端调用服务：

```go
file, err := client.OpenFile("text.txt")
if err != nil{
panic(err)
}
defer file.Close()
//....
//对file读写操作(省略)
```

不论怎么封装，最终都是落实到tcp连接的读取，我们使用chunk编码隐藏了这个过程。示意图如下：

![image.png](https://s2.loli.net/2022/03/15/dfMLIPbaD2uBW7x.png)

服务端和客户端与tcp连接直接都存在一层中间件：

+ 从tcp连接读时，自动将chunk编码数据转化为payload。

+ 往tcp连接写时，自动将payload封装成chunk编码的方式。

所以框架的用户无需手动构建或者解析chunk。

stream类型仅对rpch-go实现，其他语言正处于开发中。

# 安装

```shell
go get github.com/gufeijun/rpch-go
```

# references

+ [rpch-c](https://github.com/gufeijun/rpch-c)：rpch框架的c语言实现。
+ [rpch-go](https://github.com/gufeijun/rpch-go)：rpch框架的go语言实现。
+ [rpch-node](https://github.com/gufeijun/rpch-node)：rpch框架的nodejs实现。
