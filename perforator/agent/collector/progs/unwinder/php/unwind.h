#pragma once

#include "../process.h"
#include "types.h"

#ifdef PERFORATOR_ENABLE_PHP

static ALWAYS_INLINE bool php_retrieve_configs(
    struct process_info* proc_info,
    struct php_state* state
) {
    if (proc_info == NULL || state == NULL) {
        return false;
    }

    binary_id id = proc_info->python_binary.id;
    struct php_config* config = bpf_map_lookup_elem(&php_storage, &id);
    if (config == NULL) {
        return false;
    }

    state->config = *config;
    if (config->executor_globals_elf_vaddr != 0) {
        state->executor_globals_mem_vaddr = proc_info->php_binary.base_address + config->executor_globals_elf_vaddr;
    }

    return true;
}

static ALWAYS_INLINE void php_collect_stack(
    struct process_info* proc_info,
    struct php_state* state
) {
    if (proc_info == NULL || state == NULL || !is_mapped(proc_info->php_binary)) {
        return;
    }

    bool found_config = php_retrieve_configs(proc_info, state);
    if (!found_config) {
        return;
    }

    return;

}

#endif
