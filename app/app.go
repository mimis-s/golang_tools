package app

import (
	"fmt"

	webService "github.com/mimis-s/golang_tools/net/service"
	rpcxService "github.com/mimis-s/golang_tools/rpcx/service"
)

// app

type TaskFunc func() error

type TaskJobFunc func() error

type TaskOtherFunc func() error

type task struct {
	desc    string
	execute interface{}
}

type App struct {
	// 名字, 执行的任务(job服务,dao服务...)
	Name                   string
	bootConfig             interface{} // 每个app独有的配置信息
	serviceTasks           []task      // rpc服务
	serverTasks            []task      // web服务
	postTasks              []task      // job推送服务
	othersynchronousTasks  []task      // 其它同步服务(串行阻塞执行)
	otherasynchronousTasks []task      // 其它异步服务(协程并发执行, 不关心是否返回错误)
}

func newApp(name string) *App {
	app := &App{
		Name: name,
	}
	return app
}

func (a *App) AddService(desc string, tFunc TaskFunc) *App {
	if tFunc == nil {
		return a
	}
	a.serviceTasks = append(a.serviceTasks, task{desc: desc, execute: tFunc})
	return a
}

func (a *App) AddPost(desc string, tFunc TaskJobFunc) *App {
	if tFunc == nil {
		return a
	}
	a.postTasks = append(a.postTasks, task{desc: desc, execute: tFunc})
	return a
}

func (a *App) AddServer(desc string, tFunc TaskFunc) *App {
	if tFunc == nil {
		return a
	}
	a.serverTasks = append(a.serverTasks, task{desc: desc, execute: tFunc})
	return a
}

func (a *App) run() error {
	errChan := make(chan error, 1)
	// rpc服务
	for _, item := range a.serviceTasks {
		go func(desc string, s *rpcxService.ServerManage) {
			err := s.Run()
			if err != nil {
				errChan <- fmt.Errorf("rpc service[%v] is err:%v", desc, err)
			}
		}(item.desc, item.execute.(*rpcxService.ServerManage))
	}

	// web服务
	for _, item := range a.serverTasks {
		go func(desc string, s webService.Service) {
			err := s.Run()
			if err != nil {
				errChan <- fmt.Errorf("web service[%v] is err:%v", desc, err)
			}
		}(item.desc, item.execute.(webService.Service))
	}

	// post job服务
	for _, item := range a.postTasks {
		err := item.execute.(TaskJobFunc)()
		if err != nil {
			errStr := fmt.Errorf("run post task %s return error:%v", item.desc, err)
			fmt.Println(errStr)
			return errStr
		}
	}

	// 同步服务
	for _, item := range a.othersynchronousTasks {
		err := item.execute.(TaskOtherFunc)()
		if err != nil {
			errStr := fmt.Errorf("run other task %s return error:%v", item.desc, err)
			fmt.Println(errStr)
			return errStr
		}
	}

	// 异步服务
	for _, item := range a.otherasynchronousTasks {
		go item.execute.(TaskOtherFunc)()
	}

	fmt.Printf("app[%v] run is ok\n", a.Name)

	select {
	case err := <-errChan:
		errStr := fmt.Errorf("app[%v] run is error:%v", a.Name, err)
		fmt.Println(errStr)
		return errStr
	}

}
func (a *App) stop() {
	// rpc 服务
	for _, s := range a.serviceTasks {
		s.execute.(*rpcxService.ServerManage).Stop()
	}

	// web服务
	for _, s := range a.serverTasks {
		s.execute.(webService.Service).Stop()
	}
}
