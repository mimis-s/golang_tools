package app

// app不直接与外部进行连接, 外部只能得到app_out_side, 在register里面才能通过out_side去操作app
type AppInitFunc func(app *App) error

type AppOutSideInfo struct {
	name     string
	initFunc AppInitFunc // 留给外部的注册回调函数, 在该函数里面可以得到app进行操作
	options  []AppOption
}

func NewAppOutSide(name string, initFunc AppInitFunc) *AppOutSideInfo {
	return &AppOutSideInfo{
		name:     name,
		initFunc: initFunc,
	}
}

func (a *AppOutSideInfo) AddOptions(options ...AppOption) *AppOutSideInfo {
	a.options = append(a.options, options...)
	return a
}

func (a *AppOutSideInfo) AddAppBootConfig(appConfig interface{}) AppOption {
	return appOptionFunction(func(a *App) {
		a.bootConfig = appConfig
	})
}
