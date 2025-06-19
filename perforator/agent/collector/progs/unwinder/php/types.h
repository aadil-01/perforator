#pragma once

#include "../binary.h"


struct php_config {
    u32 version;
    u64 executor_globals_elf_vaddr;
};

#ifdef PERFORATOR_ENABLE_PHP

enum {
    PHP_MAX_STACK_DEPTH = 128,
    MAX_PHP_BINARIES = MAX_BINARIES,
};

struct php_state {
    struct php_config config;

    u64 executor_globals_mem_vaddr;

};

#else

enum {
    MAX_PHP_BINARIES = 1
};

#endif

BPF_MAP(php_storage, BPF_MAP_TYPE_HASH, binary_id, struct php_config, MAX_PHP_BINARIES);
