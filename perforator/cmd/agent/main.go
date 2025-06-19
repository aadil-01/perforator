package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/library/go/core/log"
	logzap "github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/library/go/core/log/zap/asynczap"
	"github.com/yandex/perforator/library/go/core/log/zap/encoders"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/internal/buildinfo/cobrabuildinfo"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/maxprocs"
	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	rootCmd = &cobra.Command{
		Use:           "agent",
		Short:         "Gather performance profiles and send them to storage",
		Long:          "Profiling agent tracing different cgroups' processes, sending profiles and binaries to storage",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, args []string) error {
			return run()
		},
	}

	dumpElf          bool
	debug            bool
	configPath       string
	cgroupConfigPath string
	cgroups          []string
	pids             []int
	tids             []int
	logLevel         string
	enableJVM        bool
	enablePHP        bool
)

func init() {
	rootCmd.Flags().BoolVarP(&dumpElf, "dumpelf", "d", false, "dump eBPF ELF to stdout and exit")
	rootCmd.Flags().BoolVarP(&debug, "debug", "D", false, "force debug mode")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to profiler config")
	rootCmd.Flags().StringVar(&cgroupConfigPath, "cgroups", "", "path to cgroups config")
	rootCmd.Flags().StringSliceVarP(&cgroups, "cgroup", "G", nil, "name of cgroup to trace")
	rootCmd.Flags().IntSliceVarP(&pids, "pid", "p", nil, "id of process(es) to trace")
	rootCmd.Flags().IntSliceVarP(&tids, "tid", "t", nil, "id of thread(s) to trace")
	rootCmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "log level (default - `info`, must be one of `debug`, `info`, `warn`, `error`)")
	rootCmd.Flags().BoolVar(&enableJVM, "enable-jvm", false, "[experimental feature] enable JVM profiling")
	rootCmd.Flags().BoolVar(&enablePHP, "enable-php", false, "[experimental feature] enable PHP profiling")

	cobrabuildinfo.Init(rootCmd)

	must.Must(rootCmd.MarkFlagFilename("config"))
	rootCmd.MarkFlagsOneRequired("dumpelf", "config")
	must.Must(rootCmd.MarkFlagFilename("cgroups"))
}

func main() {
	maxprocs.Adjust()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}

type CgroupsConfig struct {
	Cgroups []*profiler.CgroupConfig `yaml:"cgroups"`
}

func parseYaml(l log.Logger, path string, conf interface{}) error {
	if path == "" {
		l.Warn("No config file specified, using default")
		return nil
	}

	l.Info("Loading config file", log.String("path", path))
	configFile, err := os.Open(path)
	if err != nil {
		return err
	}

	yamlConfString, err := io.ReadAll(configFile)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(yamlConfString, conf)
}

func run() error {
	if dumpElf {
		reqs := unwinder.ProgramRequirements{
			Debug: debug,
			JVM:   enableJVM,
			PHP:   enablePHP,
		}
		prog, err := unwinder.LoadProg(reqs)
		if err != nil {
			return fmt.Errorf("failed to load program: %w", err)
		}
		_, err = io.Copy(os.Stdout, bytes.NewReader(prog))
		return err
	}

	logLevelZap, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	l, stop, err := newLogger(logLevelZap)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer stop()

	r := xmetrics.NewRegistry(
		xmetrics.WithAddCollectors(xmetrics.GetCollectFuncs()...),
		xmetrics.WithFormat(xmetrics.FormatText),
	)

	c := &config.Config{}
	err = parseYaml(l, configPath, c)
	if err != nil {
		return err
	}
	if debug {
		c.Debug = debug
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to detect hostname: %w", err)
	}

	cgroupsConfig := &CgroupsConfig{}
	if cgroupConfigPath != "" {
		err = parseYaml(l, cgroupConfigPath, cgroupsConfig)
		if err != nil {
			return err
		}
	}

	for _, cgroup := range cgroups {
		cgroupsConfig.Cgroups = append(cgroupsConfig.Cgroups, &profiler.CgroupConfig{
			Name: cgroup,
			Labels: map[string]string{
				"host": hostname,
			},
		})
	}

	p, err := profiler.NewProfiler(c, l, r)
	if err != nil {
		return err
	}

	err = p.TraceSelf(map[string]string{
		"service": "perforator",
		"host":    hostname,
	})
	if err != nil {
		return err
	}

	err = p.TraceCgroups(cgroupsConfig.Cgroups)
	if err != nil {
		return err
	}

	for _, pid := range pids {
		l.Info("Register pid", log.Int("pid", pid))
		err := p.TracePid(linux.ProcessID(pid), map[string]string{
			"host": hostname,
		})
		if err != nil {
			return fmt.Errorf("failed to start pid %d tracing: %w", pid, err)
		}
	}

	for _, tid := range tids {
		l.Info("Register tid", log.Int("tid", tid))
		err := p.TracePid(linux.ProcessID(tid), map[string]string{
			"host": hostname,
		})
		if err != nil {
			return fmt.Errorf("failed to start pid %d tracing: %w", tid, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		tick := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}

			if _, err := os.Stat("perforator.debug"); err == nil {
				err = p.SetDebugMode(true)
				if err != nil {
					l.Error("Failed to enable debug mode", log.Error(err))
				}
			} else {
				err = p.SetDebugMode(false)
				if err != nil {
					l.Error("Failed to disable debug mode", log.Error(err))
				}
			}
		}
	}()

	// Setup http puller server
	http.Handle("/metrics", r.HTTPHandler(ctx, xlog.New(l)))

	// Run pprof server
	go func() {
		err := http.ListenAndServe(":9156", nil)
		if err != nil {
			l.Error("Failed to run http server", log.Error(err))
		}
	}()

	return p.Run(ctx)
}

func newLogger(level zapcore.Level) (l log.Logger, stop func(), err error) {
	encoderconf := zap.NewProductionEncoderConfig()
	encoderconf.EncodeTime = zapcore.RFC3339NanoTimeEncoder
	encoder, err := encoders.NewTSKVEncoder(encoderconf)
	if err != nil {
		return nil, nil, err
	}

	core := asynczap.NewCore(encoder, zapcore.Lock(os.Stdout), level, asynczap.Options{
		FlushInterval: time.Second,
	})

	return logzap.NewWithCore(core), core.Stop, nil
}

var prometheusMetricSanitizer = strings.NewReplacer(
	".", "_",
	"-", "_",
)

func sanitizePrometheusMetricName(name string) string {
	return prometheusMetricSanitizer.Replace(name)
}
