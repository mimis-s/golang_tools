package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mimis-s/golang_tools/app/flags"
)

// 注册器,注册和协调app
type Registry struct {
	cmdParameterTable *GlobalCmdFlag    // 命令行启服配置参数
	customCmdFlag     interface{}       // 自定义命令行启服参数
	bootConfigFile    interface{}       // 启服配置文件
	initTasks         []task            // 在读取完配置之后做的一些初始化活动(例如:初始化csv配置表)
	appOutSides       []*AppOutSideInfo // 外部拿到和交给注册表的app信息
	apps              []*App            // 由appOutSides生成的app, 不直接和外部交流
}

/*********************外部调用***************************/
func (r *Registry) AddInitTask(desc string, tFunc TaskFunc) *Registry {
	if tFunc == nil {
		return r
	}
	r.initTasks = append(r.initTasks, task{desc: desc, execute: tFunc})
	return r
}

func (r *Registry) GetCmdParameterTable() interface{} {
	return r.cmdParameterTable
}

func (r *Registry) GetCustomCmdFlag() interface{} {
	return r.customCmdFlag
}

func (r *Registry) GetBootConfigFile() interface{} {
	return r.bootConfigFile
}

func NewRegistry(options ...RegistryOption) *Registry {
	r := &Registry{}
	r.applyOptions(options...)
	return r
}

// 外部调用为注册表增加一个app信息
func (r *Registry) AddAppOutSide(appOutSides []*AppOutSideInfo) *Registry {
	r.appOutSides = append(r.appOutSides, appOutSides...)
	return r
}

func (r *Registry) Run() error {

	// 初始化app, 日志, 监控等服务
	err := r.initialize()
	if err != nil {
		return err
	}

	// 初始化注册表的前置任务
	for _, j := range r.initTasks {
		err := j.execute.(TaskFunc)()
		if err != nil {
			err = fmt.Errorf("registry run initialize task(%s) return err:%v", j.desc, err)
			return err
		}
	}

	// 初始化app
	err = r.initApps()
	if err != nil {
		return err
	}

	// 启动调度器
	type errInfo struct {
		desc string
		scd  *App
		err  error
	}

	errChan := make(chan errInfo, 1)
	for _, app := range r.apps {
		go func(app *App) {
			err := app.run()
			if err != nil {
				// 返回调度器的报错
				errChan <- errInfo{app.Name, app, err}
			}
		}(app)
	}
	defer r.Stop()

	watchSignChan := make(chan os.Signal, 1)
	signal.Notify(watchSignChan, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)

	select {
	case signal := <-watchSignChan:
		errStr := fmt.Errorf("app receive signal(%v), will graceful stop", signal)
		fmt.Println(errStr)
		return nil
	case errInfo := <-errChan:
		errStr := fmt.Errorf("app receive register(%v) stop with error:%v", errInfo.desc, errInfo.err)
		fmt.Println(errStr)
		return err
	}
}

func (r *Registry) Stop() {
	for _, app := range r.apps {
		app.stop()
	}
}

/*********************内部调用***************************/

// 初始化
func (r *Registry) initialize() error {
	var bootConfigs []interface{}
	for _, outSide := range r.appOutSides {
		a := newApp(outSide.name)
		r.apps = append(r.apps, a)
		if a.bootConfig != nil {
			bootConfigs = append(bootConfigs, a.bootConfig)
		}
	}

	// 解析命令行参数
	flags.ParseWithStructPointers(append([]interface{}{r.cmdParameterTable, r.customCmdFlag}, bootConfigs))

	// 解析配置文件
	if r.bootConfigFile != nil && r.cmdParameterTable.BootConfigFile != "" {
		err := flags.ParseWithConfigFileContent(r.cmdParameterTable.BootConfigFile, r.bootConfigFile)
		if err != nil {
			errStr := fmt.Errorf("parse With Config File Content is error:%v", err)
			return errStr
		}
	}
	// 初始化日志系统(暂时没有接入)

	// 初始化go pprof(暂时没有接入)

	// 初始化prometheus(暂时没有接入)

	// 初始化holmes dump(暂时没有接入)
	return nil
}

// 初始化app
func (r *Registry) initApps() error {
	for i, app := range r.apps {
		if r.appOutSides[i].initFunc != nil {
			err := r.appOutSides[i].initFunc(app)
			if err != nil {
				return fmt.Errorf("app[%v] init return error[%v]", app.Name, err)
			} else {
				fmt.Printf("app[%v] initialize ok\n", app.Name)
			}
		}
	}

	return nil
}

func (scd *Registry) applyOptions(options ...RegistryOption) *Registry {
	for _, option := range options {
		option.Apply(scd)
	}
	return scd
}
