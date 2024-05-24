#include "meminfo.h"
#include <windows.h>
#include <psapi.h>
#include <iostream>

std::uint32_t meminfo::getUsage() {
    PROCESS_MEMORY_COUNTERS pmc;
    if (GetProcessMemoryInfo(GetCurrentProcess(), &pmc, sizeof(pmc)))
        return static_cast<std::uint32_t>(pmc.WorkingSetSize);
    return 0;
}
