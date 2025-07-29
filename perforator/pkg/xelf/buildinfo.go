package xelf

import (
	"debug/elf"
	"errors"
	"io"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////

type BuildInfo struct {
	BuildID      string
	LoadBias     uint64
	FirstPhdr    *elf.ProgHeader
	HasDebugInfo bool
}

////////////////////////////////////////////////////////////////////////////////

func ReadBuildInfo(r io.ReaderAt) (*BuildInfo, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var bi BuildInfo

	bi.BuildID, err = parseBuildID(f)
	if err != nil {
		return nil, err
	}

	phdr, err := parseFirstExecutableLoadablePhdr(f)
	if err != nil {
		return nil, err
	}

	if phdr != nil {
		// See https://refspecs.linuxbase.org/elf/gabi4+/ch5.pheader.html
		// "Otherwise, p_align should be a positive, integral power of 2, and p_vaddr should equal p_offset, modulo p_align"
		if phdr.Align > 1 && phdr.Vaddr%phdr.Align != phdr.Off%phdr.Align {
			return nil, errors.New("program header alignment invariant is violated")
		}

		bi.LoadBias = calculateLoadBias(phdr)
	}

	bi.FirstPhdr = parseFirstLoadablePhdrInfo(f)

	bi.HasDebugInfo, err = hasDebugInfo(f)
	if err != nil {
		return nil, err
	}

	return &bi, nil
}

func isLoadablePhdr(p *elf.Prog) bool {
	return p.Type == elf.PT_LOAD
}

func isExecutablePhdr(p *elf.Prog) bool {
	return p.Flags&elf.PF_X == elf.PF_X
}

////////////////////////////////////////////////////////////////////////////////

func calculateLoadBias(firstExecutableLoadablePhdr *elf.ProgHeader) uint64 {
	if firstExecutableLoadablePhdr == nil {
		return 0
	}

	return firstExecutableLoadablePhdr.Vaddr & ^(firstExecutableLoadablePhdr.Align - 1)
}

func parseFirstExecutableLoadablePhdr(f *elf.File) (*elf.ProgHeader, error) {
	for _, p := range f.Progs {
		if p.Filesz == 0 {
			continue
		}

		if isLoadablePhdr(p) && isExecutablePhdr(p) {
			return &p.ProgHeader, nil
		}
	}

	return nil, nil
}

func parseFirstLoadablePhdrInfo(f *elf.File) *elf.ProgHeader {
	for _, p := range f.Progs {
		if !isLoadablePhdr(p) {
			continue
		}

		return &p.ProgHeader
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

func hasDebugInfo(f *elf.File) (bool, error) {
	for _, scn := range f.Sections {
		if strings.HasPrefix(scn.Name, ".debug") || strings.HasPrefix(scn.Name, ".zdebug") {
			return true, nil
		}
	}

	return false, nil
}

////////////////////////////////////////////////////////////////////////////////
