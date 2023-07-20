package app

var ProGlobalBootFlag = &GlobalCmdFlag{}

// cmd命令行参数
type GlobalCmdFlag struct {
	GlobalID       string `flag:"global_id" desc:"全局唯一id，为空会给随机字符串" default:""`
	ServiceName    string `flag:"service_name" desc:"当前进程服务名，为空会用当前可执行文件名" default:""`
	BootConfigFile string `flag:"boot_config_file" desc:"起服配置文件路径，例如：/dir/boot_config.yaml" default:""`
	// TracePort      string `flag:"trace_port" desc:"监控端口，包含prometheus、go pprof等" default:"7788"`
	// LogDirPath     string `flag:"log_dir" desc:"程序日志输出目录，为空默认输出到控制台" default:""`
	// LogStdout      bool   `flag:"log_stdout" desc:"log_dir不为空时控制是否输出到控制台，即双份输出" default:"false"`
	// LogLevel       string `flag:"log_level" desc:"trace|debug|info|notice|warn|error|criti|fatal|panic" default:""`
}
