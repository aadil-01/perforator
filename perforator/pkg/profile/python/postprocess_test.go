package python

import (
	"slices"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

func createSimpleLocationNative(funcName string, isKernel bool) *pprof.Location {
	loc := &pprof.Location{
		Line: []pprof.Line{
			{
				Function: &pprof.Function{
					Name: funcName,
				},
			},
		},
	}
	if isKernel {
		loc.Mapping = &pprof.Mapping{File: string(profile.KernelSpecialMapping)}
	}

	return loc
}

func createSimpleLocationKernel(funcName string) *pprof.Location {
	return createSimpleLocationNative(funcName, true)
}

func createSimpleLocationUserspace(funcName string) *pprof.Location {
	return createSimpleLocationNative(funcName, false)
}

func createSimpleLocationPython(funcName string) *pprof.Location {
	loc := &pprof.Location{
		Mapping: &pprof.Mapping{File: string(profile.PythonSpecialMapping)},
		Line: []pprof.Line{
			{
				Function: &pprof.Function{
					Name: funcName,
				},
			},
		},
	}

	return loc
}

func TestMergeStacks_Simple(t *testing.T) {
	merger := NewNativeAndPythonStackMerger()

	for _, test := range []struct {
		name   string
		sample *pprof.Sample
		// if resultSample is nil, then we expect that the sample is not changed
		resultSample   *pprof.Sample
		performedMerge bool
		containsPython bool
	}{
		{
			name: "busyloop_release",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "busyloop2_release",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "busyloop1_debug",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("Py_XDECREF"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("Py_XDECREF"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "busyloop2_debug",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("_Py_DECREF_SPECIALIZED"),
					createSimpleLocationUserspace("_PyInterpreterState_GET"),
					createSimpleLocationUserspace("_PyThreadState_GET"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("_Py_DECREF_SPECIALIZED"),
					createSimpleLocationUserspace("_PyInterpreterState_GET"),
					createSimpleLocationUserspace("_PyThreadState_GET"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "only_native",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
				},
			},
			performedMerge: false,
			containsPython: false,
		},
		{
			name: "incorrect",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: false,
			containsPython: true,
		},
		{
			name: "trim_last_cpython_substack",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "python_stack_before_kernel_stack_on_failed_merge",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: false,
			containsPython: true,
		},
		{
			name: "one_to_one_python_frame_to_pyeval_cpython_3_10",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "one_to_one_python_frame_to_pyeval_cpython_3_2",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "one_to_one_python_frame_to_pyeval_cpython_3_2_one_more_py_eval",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			slices.Reverse(test.sample.Location)

			originalSample := &pprof.Sample{
				Location: make([]*pprof.Location, len(test.sample.Location)),
			}
			copy(originalSample.Location, test.sample.Location)

			if test.resultSample != nil {
				slices.Reverse(test.resultSample.Location)
			}
			stats, err := merger.MergeStacks(test.sample)
			require.NoError(t, err)

			require.Equal(t, test.performedMerge, stats.PerformedMerge, "Did not perform merge")
			require.Equal(t, test.containsPython, stats.CollectedPython, "Did not collect python")

			diffSample := originalSample
			if test.resultSample != nil {
				diffSample = test.resultSample
			}

			require.Equal(t, len(diffSample.Location), len(test.sample.Location))
			for i := 0; i < len(diffSample.Location); i++ {
				require.Equal(t, diffSample.Location[i], test.sample.Location[i])
			}
		})
	}
}
