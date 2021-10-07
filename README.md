# 前言

rpch是分布式系统与中间件课程的大作业的rpc框架部分。

在决定开发自己的rpc框架前，在对目前开源界出名的几个rpc框架体验过后，我发现它们的使用都不太符合我个人的习惯或者有其不合理不方便的部分。

> 无强类型约束

比如在国人开源的rpcx框架以及golang的标准库net/rpc中，客户端想要远程调用一个服务，需要使用类似于以下的方式：

```go
type TwoNums struct{
    A int
    B int
}
var reply int
nums := &TwuNums{A:1,B:2}
//func(*Client) Call(serviceMethodName string, args interface{}, reply interface{})
client.Call("Math.Add", args, &reply)
```

以字符串方式指定一个服务，存在着极大的拼写出错可能，无法保证在编译期间就发现错误。难道就不能直接为我们的client绑定一个Add方法，变成类似于下面的方式调用，这样在编译期间就能规避这一类错误。

```go
client.Add(args,&reply)
```

除此之外，对于服务端提供的一个服务，其欲接受的参数和返回的类型都是明确的，但客户端Call方法的传入参数args和传出参数reply都是空接口类型，无法进行有效的类型限制，程序员需要自己根据服务定义的规范来约束自己的传参和反参。但如果我们上面的Add方法签名直接就给你约束参数类型，这样是不是就可以可以大大减少程序员心智负担：

```go
//func(*Client) Add(args *TwoNums, reply *int)
```

能在编译期间就把大部分错误给规避掉，才能提高我们程序的健壮性。

很遗憾的是上面提及的框架都没有做到我说的这些，不是它们不想做，而是做不了。因为上面提到的几点功能都是和用户提供的具体服务紧密耦合的，框架开发者根本无法预支用户要暴露哪些服务，谈何为其绑定具体的服务调用方法以及定义具体的参数类型？

> protobuf的服务只能有一个传入参数，且不能是基本类型

google推出的grpc框架很好的解决上面的问题，它使用了一个中立接口描述语言(IDL)：protobuf去定义用户的rpc服务接口，然后由其编译器protoc编译这个IDL，生成到具体的某个语言的源代码文件，这些自动生成的代码就完成了上述的一些约束工作，暴露给用户的是明确的强约束的API方法。

但grpc依旧有其不便之处，如一个服务必须只能有一个传入参数和传出参数，且每个参数都必须是一个message不能是基本类型。

如我们提供一个Add服务：

```protobuf
service Math{
    rpc Add(Request) returns (Response) {}
}

message Request{
    int32 a = 1;
    int32 b = 2;
}

message Response{
	int32 result = 1;
}
```

我们要将加法的两个操作数组合成一个message: Request ，返回值就算是返回一个in32，也必须给其封装成一个message: Response，极其不方便。直接写成这样不好吗：

```protobuf
service Math{
    rpc Add(int32,int32) returns (int32) {}
}
```

> 缺少字节流的上传下载支持

除此之外，大部分rpc框架都缺少对字节流上传下载的支持，如实现一个文件读写、上传、下载服务器的rpc版本，需要做很多额外的工作如分包组包，所以我给我的框架增加了一个stream类型，熟悉protobuf的朋友可能对stream挺熟悉，但我们提供的流与protobuf的流具有本质的区别，不要混淆。最直观的感受就是看代码：

```go
//服务端代码：
func (*fileService) OpenFile(filepath string) (stream io.ReadWriter, onFinish func(), err error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return
	}
	return file, func() {
		file.Close()
	}, nil
}

//客户端代码：
file, _ := client.OpenFile("test.txt")
defer file.Close()
ioutil.ReadAll(file)
file.Write([]byte("hello world\n"))
```

服务端提供了一个RPC服务，它的用途是将服务端主机上的一个文件句柄返回给客户端。客户端能够宛如操作本机的一个文件一样直接读写服务端的文件句柄。你可能觉得这个不可思议，但这个确实是能在我的框架中实际运行的代码。我觉得这个带来的好处是很明显的，两端想读写对端的Reader或者Writer，根本无需读取出数据再传输，直接就可以把你的流句柄交给对面，省略掉很多不必要的步骤。虽然流类型的使用有一些限制，但只有使用得当，绝对能巨大提高开发效率。

# 介绍

我自己开发的rpc框架中重点解决了上面的问题。我设计了个简单的类似于protobuf这样的IDL语言，并为其写了个编译器名叫hgen(取这个名字是因为母校华科首字母就是H)，如果想体验rpch框架，请先了解编译器hgen的使用以及IDL的语法，详见[hgen](https://github.com/gufeijun/hgen)。

该仓库下的[examples](https://github.com/gufeijun/rpch-go/tree/master/examples)就是使用案例，目前在不停的增添中。欢迎您PR提交更多的案例。

启动一个最简单的rpch服务的步骤如下：

**创建IDL：**

```protobuf
//math.gfj
service Math{
    int32 Add(int32,int32)
}
```

编译器编译：`hgen -dir gfj math.gfj `。即会生成一个gfj/math.rpch.go文件，gfj是自动生成代码所在的包，其中包括了服务注册以及客户端调用服务的一些方法。

**server:**

```go
//自动生成的gfj/math.rpch.go：
//func RegisterMathService(impl MathService, svr *rpch.Server) 
//type MathService interface{
//	Add(int32, int32) (int32, error)
//}

type mathService struct{}

//实现MathService接口
func (*mathService) Add(a int32, b int32) (int32, error) {
	return a + b, nil
}

s := rpch.NewServer()
gfj.RegisterMathService(new(mathService),svr)	//此函数由编译器生成
svr.ListenAndServe("tcp","127.0.0.1:8080")
```

**client**：

```go
conn, _ := rpch.NewClient(addr)
client := gfj.NewMathServiceClient(conn)	//此函数由编译器生成
result, _ := client.Add(2, 3)

//自动生成的gfj/math.rpch.go:
//func (c *MathServiceClient) Add(arg1 int32, arg2 int32) (res int32, err error) 
```

# 安装

```shell
go get github.com/gufeijun/rpch-go
```

# TODO

+ 客户端异步调用支持

+ 支持更多语言
+ 支持udp、http协议
+ 支持服务注册以及服务发现

