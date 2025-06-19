package agent

import (
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/php"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

var (
	minSupportedVersion = encodeVersion(&php.PhpVersion{
		Major:   7,
		Minor:   4,
		Release: 0,
	})
	maxSupportedVersion = encodeVersion(&php.PhpVersion{
		Major:   7,
		Minor:   4,
		Release: 0,
	})
)

func IsVersionSupported(version *php.PhpVersion) bool {
	if version == nil {
		return false
	}

	versionKey := encodeVersion(version)
	if versionKey < minSupportedVersion || versionKey > maxSupportedVersion {
		return false
	}
	// _, ok := phpVersionOffsets[versionKey]
	return true
}

const (
	// https://github.com/php/php-src/blob/2e2494fbef842171257b0ae2b6d4392ba303f43f/Zend/zend_vm_opcodes.h#L31
	zendVmKindHybrid = 4
)

func IsSupportedZendVmKind(zendVmKind uint32) bool {
	return zendVmKind == zendVmKindHybrid
}

func encodeVersion(version *php.PhpVersion) uint32 {
	return version.Release | (version.Minor << 8) | (version.Major << 16)
}

func ParsePhpUnwinderConfig(config *php.PhpConfig) *unwinder.PhpConfig {
	return &unwinder.PhpConfig{Version: encodeVersion(config.Version), ExecutorGlobalsElfVaddr: config.ExecutorGlobalsELFVaddr}
}
