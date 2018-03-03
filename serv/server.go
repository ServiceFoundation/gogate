package serv

import (
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/wanghongfei/gogate/serv/filter"
)

type Server struct {
	host			string
	port			int

	// URI路由组件
	router 			*Router

	preFilters		[]filter.PreFilterFunc
	postFilters		[]filter.PostFilterFunc

	// fasthttp对象
	fastServ		*fasthttp.Server

	// 保存每个instanceId对应的Http Client
	proxyClients	*sync.Map

	// 保存服务地址
	// key: 服务名
	// val: host:port数组
	registryMap		*sync.Map
}

// 默认最大连接数
const MAX_CONNECTION = 5000

/*
* 创建网关服务对象
*
* PARAMS:
*	- host: 主机名(ip)
*	- port: 端口
*	- routePath: 路由配置文件路径
*	- maxConn: 最大连接数, 0表示使用默认值
*
*/
func NewGatewayServer(host string, port int, routePath string, maxConn int) (*Server, error) {
	if "" == host {
		return nil, errors.New("invalid host")
	}

	if port <= 0 || port > 65535 {
		return nil, errors.New("invalid port")
	}

	if maxConn <= 0 {
		maxConn = MAX_CONNECTION
	}

	// 创建router
	router, err := NewRouter(routePath)
	if nil != err {
		return nil, err
	}

	// 创建Server对象
	serv := &Server{
		host: host,
		port: port,

		router: router,
		proxyClients: new(sync.Map),
	}

	// 创建FastServer对象
	fastServ := &fasthttp.Server{
		Concurrency: maxConn,
		Handler: serv.HandleRequest,
	}

	serv.fastServ = fastServ

	// 注册过虑器
	serv.RegisterPreFilter(filter.ServiceMatchPreFilter)

	return serv, nil

}

func (s *Server) Start() error {
	return s.fastServ.ListenAndServe(s.host + ":" + strconv.Itoa(s.port))
}

func (s *Server) Shutdown() {
	// todo gracefully shutdown
}

func (s *Server) ReloadRoute() error {
	return s.router.ReloadRoute()
}

func (s *Server) ExtractRoute() string {
	return s.router.ExtractRoute()
}

func (s *Server) RegisterPreFilter(preFunc filter.PreFilterFunc) {
	s.preFilters = append(s.preFilters, preFunc)
}

func (s *Server) RegisterPostFilter(postFunc filter.PostFilterFunc) {
	s.postFilters = append(s.postFilters, postFunc)
}

func (s *Server) startRefreshRegistryInfo() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		err := refreshRegistry(s)
		if nil != err {
			log.Println(err)
		}

		<- ticker.C
	}()
}



