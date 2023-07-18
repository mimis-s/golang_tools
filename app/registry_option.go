package app

type RegistryOption interface {
	Apply(app *Registry)
}

type registryOptionFun func(r *Registry)

func (of registryOptionFun) Apply(r *Registry) {
	of(r)
}

// 命令行启服配置参数
func AddRegistryGlobalCmdFlag(f *GlobalCmdFlag) RegistryOption {
	return registryOptionFun(func(r *Registry) {
		r.cmdParameterTable = f
	})
}

// 设置启动配置文件的解析结构，不设置默认无起服配置，默认以yaml解析
func AddRegistryBootConfigFile(content interface{}) RegistryOption {
	return registryOptionFun(func(r *Registry) {
		r.bootConfigFile = content
	})
}

// 设置自定义启服参数
func AddRegistryExBootFlags(content interface{}) RegistryOption {
	return registryOptionFun(func(r *Registry) {
		r.CustomCmdFlag = content
	})
}
