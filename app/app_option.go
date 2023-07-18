package app

type AppOption interface {
	Apply(a *App)
}

type appOptionFunction func(a *App)

func (of appOptionFunction) Apply(a *App) {
	of(a)
}
