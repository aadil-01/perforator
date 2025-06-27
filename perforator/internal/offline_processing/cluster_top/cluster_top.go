package cluster_top

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ClusterTop struct {
	l xlog.Logger

	downloader *downloader.Downloader

	profileStorage profile.Storage

	symbolizer *ClusterTopSymbolizer
}

func NewClusterTop(
	conf *Config,
	l xlog.Logger,
	reg xmetrics.Registry,
	storageBundle *bundle.StorageBundle,
) (*ClusterTop, error) {
	fileCache, err := asyncfilecache.NewFileCache(
		conf.BinaryProvider.FileCache,
		l,
		reg,
	)
	if err != nil {
		return nil, err
	}

	downloaderInstance, err := downloader.NewDownloader(
		l.WithName("Downloader"),
		reg,
		fileCache,
		downloader.Config{
			MaxSimultaneousDownloads: uint64(conf.BinaryProvider.MaxSimultaneousDownloads),
		},
	)
	if err != nil {
		return nil, err
	}

	gsymDownloader, err := downloader.NewGSYMDownloader(downloaderInstance, storageBundle.BinaryStorage.GSYM())
	if err != nil {
		return nil, err
	}

	symbolizer, err := NewClusterTopSymbolizer(l, gsymDownloader)
	if err != nil {
		return nil, err
	}

	return &ClusterTop{
		l: l,

		downloader: downloaderInstance,

		profileStorage: storageBundle.ProfileStorage,

		symbolizer: symbolizer,
	}, nil
}

func buildSelector(serviceName string, timeRange TimeRange) (*querylang.Selector, error) {
	selectorStr := fmt.Sprintf("{%s=\"%s\", %s=\"%s\", %s=\"%s\"}",
		profilequerylang.EventTypeLabel, sampletype.SampleTypeCPUCycles,
		profilequerylang.ServiceLabel, serviceName,
		profilequerylang.SystemNameLabel, "perforator",
	)

	selector, err := profilequerylang.ParseSelector(selectorStr)
	if err != nil {
		return nil, err
	}

	selector.Matchers = append(
		selector.Matchers,
		profilequerylang.BuildMatcher(
			profilequerylang.TimestampLabel,
			querylang.AND,
			querylang.Condition{Operator: operator.GTE},
			[]string{timeRange.From.Format(time.RFC3339Nano)},
		),
	)

	selector.Matchers = append(
		selector.Matchers,
		profilequerylang.BuildMatcher(
			profilequerylang.TimestampLabel,
			querylang.AND,
			querylang.Condition{Operator: operator.LT},
			[]string{timeRange.To.Format(time.RFC3339Nano)},
		),
	)

	return selector, nil
}

const kDegreeOfParallelism int = 4
const kProfilesBatchSize int = 200

func (t *ClusterTop) Run(
	ctx context.Context,
	serviceSelector ServiceSelector,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := t.downloader.RunBackgroundDownloader(ctx)
		if err != nil {
			t.l.Error(ctx, "Failed background downloader", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		aggregateG, ctx := errgroup.WithContext(ctx)

		for range kDegreeOfParallelism {
			aggregateG.Go(func() error {
				for {
					if !t.selectAndProcessService(ctx, serviceSelector, clusterPerfTopAggregator) {
						if ctx.Err() != nil {
							break
						}

						time.Sleep(10 * time.Second)
					}
				}

				return nil
			})
		}

		err := aggregateG.Wait()
		if err != nil {
			return err
		}

		return nil
	})

	return g.Wait()
}

func (t *ClusterTop) selectAndProcessService(
	ctx context.Context,
	serviceSelector ServiceSelector,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
) bool {
	serviceHandler, err := serviceSelector.SelectService(ctx)
	if err != nil {
		t.l.Warn(ctx, "Failed to select a service", log.Error(err))
		return false
	}
	defer func() {
		serviceHandler.Finalize(ctx, err)
	}()

	err = t.processService(
		ctx,
		clusterPerfTopAggregator,
		serviceHandler.GetGeneration(),
		serviceHandler.GetServiceName(),
		serviceHandler.GetTimeRange(),
	)
	if err != nil {
		t.l.Error(
			ctx,
			"Failed to process the service",
			log.String("service", serviceHandler.GetServiceName()),
			log.Error(err),
		)
	}

	return true
}

func (t *ClusterTop) processService(
	ctx context.Context,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
	generation int,
	serviceName string,
	timeRange TimeRange,
) error {
	aggregator, err := t.symbolizer.NewServicePerfTopAggregator(serviceName)
	if err != nil {
		return err
	}
	defer aggregator.Destroy()

	selector, err := buildSelector(serviceName, timeRange)
	if err != nil {
		return err
	}

	profilesQuery := &meta.ProfileQuery{
		Selector: selector,
	}

	profileMetas, err := t.profileStorage.SelectProfiles(ctx, profilesQuery)
	if err != nil {
		return err
	}

	buildIDs := getBuildIDsFromProfiles(profileMetas)

	if err := aggregator.InitializeSymbolizers(ctx, buildIDs); err != nil {
		return err
	}

	t.l.Info(
		ctx,
		"New service to process",
		log.String("service", serviceName),
		log.Int("profilesCount", len(profileMetas)),
	)

	for i := 0; i < len(profileMetas); i += kProfilesBatchSize {
		metaBatch := profileMetas[i:min(i+kProfilesBatchSize, len(profileMetas))]
		batch, err := t.fetchProfiles(ctx, metaBatch)

		if err != nil {
			return err
		}

		t.l.Info(
			ctx,
			"Got a batch of profiles to process",
			log.String("service", serviceName),
			log.Int("batchSize", len(batch)),
		)

		err = aggregator.AddProfiles(ctx, batch)
		if err != nil {
			return err
		}
	}

	t.l.Info(
		ctx,
		"Finished service processing",
		log.String("service", serviceName),
	)

	functions := aggregator.Extract()

	return clusterPerfTopAggregator.Save(ctx, &ServicePerfTop{
		Generation:  generation,
		ServiceName: serviceName,
		Functions:   functions,
	})
}

func (t *ClusterTop) fetchProfiles(
	ctx context.Context,
	profileMetas []*meta.ProfileMetadata,
) ([]profile.ProfileData, error) {
	profiles := make([]profile.ProfileData, len(profileMetas))

	g, ctx := errgroup.WithContext(ctx)
	for i := range profileMetas {
		g.Go(func() error {
			noExistErr := &blob.ErrNoExist{}

			data, err := t.profileStorage.FetchProfile(ctx, profileMetas[i])
			if err != nil && !errors.As(err, &noExistErr) {
				return err
			}

			profiles[i] = data

			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		return nil, err
	}

	return profiles, nil
}

func getBuildIDsFromProfiles(profileMetas []*meta.ProfileMetadata) []string {
	uniqueBuildIDs := make(map[string]struct{})

	for _, profileMeta := range profileMetas {
		for _, buildID := range profileMeta.BuildIDs {
			uniqueBuildIDs[buildID] = struct{}{}
		}
	}

	uniqueBuildIDsList := make([]string, 0, len(uniqueBuildIDs))
	for buildID := range uniqueBuildIDs {
		uniqueBuildIDsList = append(uniqueBuildIDsList, buildID)
	}

	return uniqueBuildIDsList
}
