package cmd

import (
	"os"

	"github.com/google/pprof/profile"
	"github.com/spf13/cobra"

	"github.com/yandex/perforator/perforator/pkg/must"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
	"github.com/yandex/perforator/perforator/pkg/profile/parse/perf"
)

func makeFlamegraphCmd() *cobra.Command {
	var inputPath string
	var baselinePath string
	var minWeight = 0.000001
	var maxDepth = 0
	var format = render.HTMLFormat
	var title = "Flamegraph"
	var sampleType = "cycles"

	flamegraphPerfCmd := &cobra.Command{
		Use:   "perf",
		Short: "Render flamegraph from perf script",
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildPerfFlamegraph(inputPath, baselinePath, format, minWeight, maxDepth, title, sampleType)
		},
	}
	flamegraphPProfCmd := &cobra.Command{
		Use:   "pprof",
		Short: "Render flamegraph from pprof profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildPProfFlamegraph(inputPath, baselinePath, format, minWeight, maxDepth, title, sampleType)
		},
	}
	flamegraphCollapsedCmd := &cobra.Command{
		Use:   "collapsed",
		Short: "Render flamegraph from collapsed stacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildCollapsedFlamegraph(inputPath, baselinePath, format, minWeight, maxDepth, title, sampleType)
		},
	}

	flamegraphCmd := &cobra.Command{
		Use:   "flamegraph",
		Short: "Build flamegraph from various profiles",
	}

	flamegraphCmd.PersistentFlags().StringVarP(&inputPath, "input", "i", "stdin", "Path to the input")
	flamegraphCmd.PersistentFlags().StringVarP(&baselinePath, "baseline", "b", "", "Path to the baseline profile")
	flamegraphCmd.PersistentFlags().StringVarP((*string)(&format), "format", "f", "html", "Render format (html or html_v2)")
	flamegraphCmd.PersistentFlags().Float64VarP(&minWeight, "min-weight", "w", 0, "Minimum function weight to draw")
	flamegraphCmd.PersistentFlags().IntVarP(&maxDepth, "max-depth", "d", 0, "Maximum flamegraph height. Use 0 to disable")
	flamegraphCmd.PersistentFlags().StringVarP(&title, "title", "t", "Flamegraph", "Flamegraph title")
	flamegraphCmd.PersistentFlags().StringVarP(&sampleType, "sample-type", "T", "cycles", "Flamegraph sample type")
	must.Must(flamegraphCmd.MarkPersistentFlagFilename("input"))

	flamegraphCmd.AddCommand(flamegraphPerfCmd)
	flamegraphCmd.AddCommand(flamegraphPProfCmd)
	flamegraphCmd.AddCommand(flamegraphCollapsedCmd)

	return flamegraphCmd
}

func makeProfileCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Perform operations with local profile",
	}

	cmd.AddCommand(makeFlamegraphCmd())
	return cmd
}
func init() {
	rootCmd.AddCommand(makeProfileCommand())
}

func open(path string) (f *os.File, done func(), err error) {
	f = os.Stdin
	done = func() {}

	if path != "" && path != "stdin" {
		f, err = os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		done = func() {
			_ = f.Close()
		}
	}

	return f, done, err
}

func loadProfile(path string) (*profile.Profile, error) {
	input, done, err := open(path)
	if err != nil {
		return nil, err
	}
	defer done()

	return profile.Parse(input)
}

func loadFoldedProfile(path string) (*collapsed.Profile, error) {
	input, done, err := open(path)
	if err != nil {
		return nil, err
	}
	defer done()

	return collapsed.Decode(input)
}

func loadPerfProfile(path string) (*profile.Profile, error) {
	input, done, err := open(path)
	if err != nil {
		return nil, err
	}
	defer done()

	return perf.ParsePerfScript(input)
}

func buildPerfFlamegraph(inputPath, baselinePath string, format render.Format, minWeight float64, maxDepth int, title string, sampleType string) error {
	fg := render.NewFlameGraph()
	fg.SetTitle(title)
	fg.SetMinWeight(minWeight)
	fg.SetDepthLimit(maxDepth)
	fg.SetFormat(format)
	fg.SetSampleType(sampleType)

	prof, err := loadPerfProfile(inputPath)
	if err != nil {
		return err
	}
	err = fg.AddProfile(prof)
	if err != nil {
		return err
	}

	if baselinePath != "" {
		prof, err := loadPerfProfile(baselinePath)
		if err != nil {
			return err
		}
		err = fg.AddBaselineProfile(prof)
		if err != nil {
			return err
		}
	}

	err = fg.Render(os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func buildPProfFlamegraph(inputPath, baselinePath string, format render.Format, minWeight float64, maxDepth int, title string, sampleType string) error {
	fg := render.NewFlameGraph()
	fg.SetTitle(title)
	fg.SetMinWeight(minWeight)
	fg.SetDepthLimit(maxDepth)
	fg.SetFormat(format)
	fg.SetSampleType(sampleType)

	prof, err := loadProfile(inputPath)
	if err != nil {
		return err
	}
	err = fg.AddProfile(prof)
	if err != nil {
		return err
	}

	if baselinePath != "" {
		prof, err := loadProfile(baselinePath)
		if err != nil {
			return err
		}
		err = fg.AddBaselineProfile(prof)
		if err != nil {
			return err
		}
	}

	err = fg.Render(os.Stdout)
	if err != nil {
		return err
	}

	return nil
}

func buildCollapsedFlamegraph(inputPath, baselinePath string, format render.Format, minWeight float64, maxDepth int, title string, sampleType string) error {
	fg := render.NewFlameGraph()
	fg.SetTitle(title)
	fg.SetMinWeight(minWeight)
	fg.SetDepthLimit(maxDepth)
	fg.SetFormat(format)
	fg.SetSampleType(sampleType)

	prof, err := loadFoldedProfile(inputPath)
	if err != nil {
		return err
	}
	err = fg.AddCollapsedProfile(prof)
	if err != nil {
		return err
	}

	if baselinePath != "" {
		prof, err := loadFoldedProfile(baselinePath)
		if err != nil {
			return err
		}
		err = fg.AddCollapsedBaselineProfile(prof)
		if err != nil {
			return err
		}
	}

	err = fg.Render(os.Stdout)
	if err != nil {
		return err
	}

	return nil
}
