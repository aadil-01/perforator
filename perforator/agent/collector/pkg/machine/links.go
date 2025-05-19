package machine

import (
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux/kallsyms"
)

type bpfLinks struct {
	l log.Logger

	FinishTaskSwitch link.Link
	SignalDeliver    link.Link
}

func (l *bpfLinks) close() error {
	if l == nil {
		return nil
	}

	var errs []error
	if l.FinishTaskSwitch != nil {
		errs = append(errs, l.FinishTaskSwitch.Close())
	}
	if l.SignalDeliver != nil {
		errs = append(errs, l.SignalDeliver.Close())
	}

	return errors.Join(errs...)
}

func (l *bpfLinks) setup(conf *Config, progs *unwinder.Progs) (err error) {
	defer func() {
		if err != nil {
			closeErr := l.close()
			if closeErr != nil {
				l.l.Error("Failed to close links on failed setupLinks", log.Error(closeErr))
			}
		}
	}()

	if enabled := conf.TraceWallTime; enabled != nil && *enabled {
		l.FinishTaskSwitch, err = createKprobeBySymbolRegex(`^finish_task_switch(\.isra\.\d+)?$`, progs.PerforatorFinishTaskSwitch)
		if err != nil {
			return fmt.Errorf("failed to setup kprobe finish_task_switch link: %w", err)
		}
	}

	if enabled := conf.TraceSignals; enabled != nil && *enabled {
		l.SignalDeliver, err = link.Tracepoint("signal", "signal_deliver", progs.PerforatorSignalDeliver, nil)
		if err != nil {
			return fmt.Errorf("failed to setup tracepoint signal_deliver link: %w", err)
		}
	}

	return nil
}

func newBPFLinks(l log.Logger) *bpfLinks {
	return &bpfLinks{
		l: l,
	}
}

// See https://github.com/iovisor/bcc/pull/3315
func findKernelSymbolsByRegex(regex string) ([]string, error) {
	resolver, err := kallsyms.DefaultKallsymsResolver()
	if err != nil {
		return nil, err
	}
	symbols, err := resolver.LookupSymbolRegex(regex)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup kprobe symbol by regex %s: %w", regex, err)
	}

	return symbols, nil
}

func createKprobeBySymbolRegex(symbolRegex string, prog *ebpf.Program) (link.Link, error) {
	symbols, err := findKernelSymbolsByRegex(symbolRegex)
	if err != nil {
		return nil, err
	}

	var errs []error
	for _, symbol := range symbols {
		kprobe, err := link.Kprobe(symbol, prog, nil)
		if err == nil {
			return kprobe, nil
		}

		errs = append(errs, err)
	}

	return nil, fmt.Errorf("failed to attach kprobe by regex %s: %w", symbolRegex, errors.Join(errs...))
}
