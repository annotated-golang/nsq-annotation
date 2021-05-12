package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/judwhite/go-svc"
	"github.com/mreiferson/go-options"
	"github.com/nsqio/nsq/internal/lg"
	"github.com/nsqio/nsq/internal/version"
	"github.com/nsqio/nsq/nsqd"
)

// program 实现go-svc的接口，实现了Init,Start,Stop,Context方法。
// 可以帮助兼容Windows服务器，查看Init方法。
type program struct {
	once sync.Once
	nsqd *nsqd.NSQD
}

func main() {
	prg := &program{}
	// SIGTERM handling is in Start()
	// --------------------------------------------------
	// 运行go-svc的Run方法，实则运行 program 的Init和Start，
	// 并且等待Stop。
	if err := svc.Run(prg, syscall.SIGINT); err != nil {
		logFatal("%s", err)
	}
}

// Init go-svc接口实现方法，检查当前主机是否为Windows。如果是
// 则创建相应的日志目录。
func (p *program) Init(env svc.Environment) error {
	if env.IsWindowsService() {
		dir := filepath.Dir(os.Args[0])
		return os.Chdir(dir)
	}
	return nil
}

// Start 运行nsqd
func (p *program) Start() error {
	// 初始化全部配置
	opts := nsqd.NewOptions()

	// 读取flag替换掉初始化的配置
	flagSet := nsqdFlagSet(opts)
	flagSet.Parse(os.Args[1:])

	rand.Seed(time.Now().UTC().UnixNano())

	// 如果是 version 命令 打印version则退出
	if flagSet.Lookup("version").Value.(flag.Getter).Get().(bool) {
		fmt.Println(version.String("nsqd"))
		os.Exit(0)
	}

	// 读取配置文件里的配置项
	var cfg config
	configFile := flagSet.Lookup("config").Value.String()
	if configFile != "" {
		_, err := toml.DecodeFile(configFile, &cfg)
		if err != nil {
			logFatal("failed to load config file %s - %s", configFile, err)
		}
	}
	cfg.Validate()

	// 合并flag和配置文件，得到最终配置。
	options.Resolve(opts, flagSet, cfg)

	// 生成nsqd对象，并加载元数据
	nsqd, err := nsqd.New(opts)
	if err != nil {
		logFatal("failed to instantiate nsqd - %s", err)
	}
	p.nsqd = nsqd

	err = p.nsqd.LoadMetadata()
	if err != nil {
		logFatal("failed to load metadata - %s", err)
	}
	err = p.nsqd.PersistMetadata()
	if err != nil {
		logFatal("failed to persist metadata - %s", err)
	}

	// 监听SIGTERM信号
	signalChan := make(chan os.Signal, 1)
	go func() {
		// range over all term signals
		// we don't want to un-register our sigterm handler which would
		// cause default go behavior to apply
		for range signalChan {
			p.once.Do(func() {
				p.nsqd.Exit()
			})
		}
	}()
	signal.Notify(signalChan, syscall.SIGTERM)

	// 运行nsqd
	go func() {
		err := p.nsqd.Main()
		// 错误则自己调用Stop，并错误退出状态码
		if err != nil {
			p.Stop()
			os.Exit(1)
		}
	}()

	return nil
}

// Stop 停止nsqd程序
func (p *program) Stop() error {
	p.once.Do(func() {
		p.nsqd.Exit()
	})
	return nil
}

// Context returns a context that will be canceled when nsqd initiates the shutdown
// --------------------------------------------------------------------------------------------
// Context 返回一个context对象，当nsqd进行关闭
func (p *program) Context() context.Context {
	return p.nsqd.Context()
}

func logFatal(f string, args ...interface{}) {
	lg.LogFatal("[nsqd] ", f, args...)
}
