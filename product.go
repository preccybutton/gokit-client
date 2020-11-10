package main

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
	httptransport "github.com/go-kit/kit/transport/http"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/go-kit/kit/sd/consul"
	"goclient/service"
	"io"
	"net/url"
	"os"
)

func main2(){
	tgt , _:= url.Parse("http://10.61.2.202:8080")	//解析一个url，生成一个url对象
	//第一步：创建一个client，第一个func是如何请求，第二个是如何解析响应
	client := httptransport.NewClient("GET", tgt, Service.GetUserInfo_Request, Service.GetUserInfo_Response)
	//第二步：暴露一个endpoint，这就是一个func，直接调用
	getUserInfo := client.Endpoint()

	//第三步：创建上下文
	ctx := context.Background()
	//第四步：执行
	res, err := getUserInfo(ctx, Service.UserRequest{102, "GET"})	//第二个参数是GetUserInfo_Request里面的r
	if err != nil{
		fmt.Println(err, res)
		os.Exit(1)
	}
	//对response断言
	userinfo := res.(Service.UserResponse)
	fmt.Println(userinfo.Result)

}

func main(){
	{
		var client consul.Client
		{
			//第一步创建client
			config := consulapi.DefaultConfig()
			config.Address = "10.61.1.232:8500"

			api_client, _ := consulapi.NewClient(config)
			client = consul.NewClient(api_client)
		}
		var logger log.Logger
		{
			logger = log.NewLogfmtLogger(os.Stdout)
		}
		{
			tags := []string{"primary"}	//consul上tag为primary,一个服务可能有多个tag，我们可以通过tag选择
			//查询服务实例的状态,passingOnly设为true表示健康检查通过，服务才能获取
			instancer := consul.NewInstancer(client, logger, "userservice", tags, true)
			{
				//生成一个工厂f，不同的service_url生成不同的endpoint，这个service_url是服务地址，即ip:port
				f := func(service_url string) (endpoint.Endpoint, io.Closer, error){
					tart, _ := url.Parse("http://"+service_url)
					return httptransport.NewClient("GET", tart, Service.GetUserInfo_Request, Service.GetUserInfo_Response).Endpoint(), nil, nil
				}
				//根据consul获得的信息使用工厂f生成暴露endpoint的func
				endpointer := sd.NewEndpointer(instancer, f, logger)
				//里面的元素是暴露了某个endpoint的func
				endpoints, _ := endpointer.Endpoints()
				fmt.Printf("服务有%d条", len(endpoints))
				//获取getUserInfo，这是一个func，可直接调用

				//创建一个endpoint的负载均衡器，里面是简单的轮询负载，依次访问endpoint，到尾再从头开始,一般都是轮询
				mylb := lb.NewRoundRobin(endpointer)
				//创建一个endpoint的负载均衡器，里面是随机负载，第二个参数为随机因子，重复概率比较小
				//fmt.Println(time.Now().UnixNano())
				//mylb := lb.NewRandom(endpointer, time.Now().UnixNano())

				//下面是使用获得的endpoint发送请求
				{
					//使用负载均衡器获取一个暴露endpoint的func
					getUserInfo, err  := mylb.Endpoint()
					if err != nil{	//比如没有service会返回错误
						return
					}
					ctx := context.Background()
					//第四步：执行
					res, err := getUserInfo(ctx, Service.UserRequest{101, "GET"})	//第二个参数是GetUserInfo_Request里面的r
					if err != nil{
						fmt.Println(err, res)
						os.Exit(1)
					}
					//对response断言
					userinfo := res.(Service.UserResponse)
					fmt.Println(userinfo.Result)
				}

			}
		}
	}

}