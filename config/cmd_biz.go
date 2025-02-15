package config

import (
	"flag"
	"time"

	"github.com/banbox/banbot/core"
	"github.com/banbox/banbot/utils"
	"github.com/banbox/banexg/errs"
	"github.com/banbox/banexg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (a *CmdArgs) Init() {
	if a.Inited {
		return
	}
	a.TimeFrames = utils.SplitSolid(a.RawTimeFrames, ",", true)
	a.Pairs = utils.SplitSolid(a.RawPairs, ",", true)
	a.Tables = utils.SplitSolid(a.RawTables, ",", true)
	if a.DataDir != "" {
		DataDir = a.DataDir
	}
	if a.InPath != "" {
		a.InPath = ParsePath(a.InPath)
	}
	if a.OutPath != "" {
		a.OutPath = ParsePath(a.OutPath)
	}
	a.Inited = true
}

func (a *CmdArgs) parseTimeZone() (*time.Location, *errs.Error) {
	if a.TimeZone != "" {
		loc, err_ := time.LoadLocation(a.TimeZone)
		if err_ != nil {
			err := errs.NewMsg(errs.CodeRunTime, "unsupport timezone: %s, %v", a.TimeZone, err_)
			return nil, err
		}
		return loc, nil
	}
	return nil, nil
}

func (a *CmdArgs) SetLog(showLog bool, handlers ...zapcore.Core) {
	logCfg := &log.Config{
		Stdout:            true,
		Format:            "text",
		Level:             a.LogLevel,
		Handlers:          handlers,
		DisableStacktrace: true,
	}
	if a.LogLevel == "" {
		logCfg.Level = "info"
	}
	if a.Logfile != "" {
		logCfg.File = &log.FileLogConfig{
			LogPath: a.Logfile,
		}
	}
	log.SetupLogger(logCfg)
	core.LogFile = a.Logfile
	if showLog && a.Logfile != "" {
		log.Info("Log To", zap.String("path", a.Logfile))
	}
}

func (a *CmdArgs) BindToFlag(cmd *flag.FlagSet) {
	cmd.StringVar(&a.DataDir, "datadir", "", "Path to data dir.")
	cmd.Var(&a.Configs, "config", "config path to use, Multiple -config options may be used")
	cmd.BoolVar(&a.NoDefault, "no-default", false, "ignore default: config.yml, config.local.yml")
	cmd.StringVar(&a.Logfile, "logfile", "", "Log to the file specified")
	cmd.StringVar(&a.LogLevel, "level", "info", "set logging level to debug")
	//cmd.BoolVar(&a.NoCompress, "no-compress", false, "disable compress for hyper table")
	cmd.IntVar(&a.MaxPoolSize, "max-pool-size", 0, "max pool size for db")
}
