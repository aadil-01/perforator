package render

import (
	"bytes"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextFormatConfiguration(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		assert.Equal(t, true, tf.locationFrameOptions.FileNames)
		assert.Equal(t, "", tf.locationFrameOptions.FilePathPrefix)
		assert.Equal(t, false, tf.locationFrameOptions.LineNumbers)
	})

	t.Run("CustomConfiguration", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetLineNumbers(true)
		tf.SetFileNames(false)
		tf.SetAddressRenderPolicy(RenderAddressesAlways)

		assert.Equal(t, true, tf.locationFrameOptions.LineNumbers)
		assert.Equal(t, false, tf.locationFrameOptions.FileNames)
		assert.Equal(t, RenderAddressesAlways, tf.locationFrameOptions.AddressPolicy)
	})
}

func TestTextFormatErrorHandling(t *testing.T) {
	t.Run("NilProfile", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		var buf bytes.Buffer
		err := tf.Render(&buf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no profile to render")
	})

	t.Run("UnsupportedFormat", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetFormat("unsupported")
		err := tf.AddProfile(createTestProfile())
		assert.NoError(t, err)

		var buf bytes.Buffer
		err = tf.Render(&buf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported format")
	})
}

func TestComponentRendering(t *testing.T) {
	t.Run("WriteStackTrace", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetLineNumbers(true)
		tf.SetFileNames(true)

		loc1 := &profile.Location{
			ID:      1,
			Address: 0x1000,
			Mapping: &profile.Mapping{
				ID:   1,
				File: "binary",
			},
			Line: []profile.Line{
				{
					Function: &profile.Function{
						ID:        1,
						Name:      "function1",
						Filename:  "file1.go",
						StartLine: 10,
					},
					Line: 10,
				},
			},
		}

		loc2 := &profile.Location{
			ID:      2,
			Address: 0x2000,
			Mapping: &profile.Mapping{
				ID:   1,
				File: "binary",
			},
			Line: []profile.Line{
				{
					Function: &profile.Function{
						ID:        2,
						Name:      "function2",
						Filename:  "file2.go",
						StartLine: 20,
					},
					Line: 20,
				},
			},
		}

		sample := &profile.Sample{
			Location: []*profile.Location{loc1, loc2},
		}

		var buf bytes.Buffer
		err := tf.writeStackTrace("  ", sample, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "  1: function1 file1.go:10")
		assert.Contains(t, output, "  2: function2 file2.go:20")
	})
}

func TestLocationFrameHandling(t *testing.T) {
	t.Run("InlinedFunctions", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetLineNumbers(true)

		// Create a location with inlined functions
		loc := &profile.Location{
			ID:      1,
			Address: 0x1000,
			Mapping: &profile.Mapping{
				ID:   1,
				File: "binary",
			},
			Line: []profile.Line{
				{
					Function: &profile.Function{
						ID:        1,
						Name:      "main",
						Filename:  "main.go",
						StartLine: 10,
					},
					Line: 10,
				},
				{
					Function: &profile.Function{
						ID:        2,
						Name:      "inlined1",
						Filename:  "inline.go",
						StartLine: 20,
					},
					Line: 20,
				},
			},
		}

		frames := tf.getLocationFrames(loc)
		require.Len(t, frames, 2)

		assert.Equal(t, "inlined1", frames[0].name)
		assert.Equal(t, "inline.go:20", frames[0].file)
		assert.True(t, frames[0].inlined)

		assert.Equal(t, "main", frames[1].name)
		assert.Equal(t, "main.go:10", frames[1].file)
		assert.False(t, frames[1].inlined)
	})
}

func TestFullProfileRendering(t *testing.T) {
	t.Run("BasicProfile", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetLineNumbers(true)
		profile := createTestProfile()

		err := tf.AddProfile(profile)
		require.NoError(t, err)

		output, err := tf.RenderBytes()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "Profile Sample Type: cpu (nanoseconds)")
		assert.Contains(t, outputStr, "Sample #1:")
		assert.Contains(t, outputStr, "  Labels:")
		assert.Contains(t, outputStr, "    key1: value1")
		assert.Contains(t, outputStr, "  Stack Trace (most recent call first):")
		assert.Contains(t, outputStr, "    1: function1 test.go:10")
	})

	t.Run("WithoutLineNumbers", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetLineNumbers(false)
		profile := createTestProfile()

		err := tf.AddProfile(profile)
		require.NoError(t, err)

		output, err := tf.RenderBytes()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "    1: function1 test.go")
		assert.NotContains(t, outputStr, "test.go:10")
	})

	t.Run("WithoutFileNames", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetFileNames(false)
		profile := createTestProfile()

		err := tf.AddProfile(profile)
		require.NoError(t, err)

		output, err := tf.RenderBytes()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "    1: function1")
		assert.NotContains(t, outputStr, "test.go")
	})

	t.Run("WithAddressRendering", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		tf.SetAddressRenderPolicy(RenderAddressesAlways)
		profile := createTestProfile()

		err := tf.AddProfile(profile)
		require.NoError(t, err)

		output, err := tf.RenderBytes()
		require.NoError(t, err)

		outputStr := string(output)
		assert.Contains(t, outputStr, "{2000} function1")
	})
}

func TestRenderPProf(t *testing.T) {
	t.Run("DirectRendering", func(t *testing.T) {
		tf := NewTextFormatRenderer()
		profile := createTestProfile()

		var buf bytes.Buffer
		err := tf.RenderPProf(profile, &buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Profile Sample Type: cpu (nanoseconds)")
		assert.Contains(t, output, "Sample #1:")
	})
}

func createTestProfile() *profile.Profile {
	loc := createTestLocation(1, "function1", "test.go", 10)

	return &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "nanoseconds"},
		},
		DefaultSampleType: "cpu",
		Sample: []*profile.Sample{
			{
				Location: []*profile.Location{loc},
				Value:    []int64{1000},
				Label: map[string][]string{
					"key1": {"value1"},
				},
				NumLabel: map[string][]int64{
					"pid": {1234},
				},
			},
		},
		Location: []*profile.Location{loc},
		Function: []*profile.Function{
			{
				ID:        1,
				Name:      "function1",
				Filename:  "test.go",
				StartLine: 10,
			},
		},
		Mapping: []*profile.Mapping{
			{
				ID:   1,
				File: "binary",
			},
		},
	}
}

func createTestLocation(id uint64, funcName, filename string, line int64) *profile.Location {
	return &profile.Location{
		ID:      id,
		Address: 0x1000 + (id * 0x1000),
		Mapping: &profile.Mapping{
			ID:   1,
			File: "binary",
		},
		Line: []profile.Line{
			{
				Function: &profile.Function{
					ID:        id,
					Name:      funcName,
					Filename:  filename,
					StartLine: line,
				},
				Line: line,
			},
		},
	}
}
