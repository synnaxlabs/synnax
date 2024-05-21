#include "meminfo.h"
#include <mach/mach.h>
#include <iostream>

std::uint32_t meminfo::getUsage() {
    task_basic_info info;
    mach_msg_type_number_t infoCount = TASK_BASIC_INFO_COUNT;

    if (task_info(
            mach_task_self(),
            TASK_BASIC_INFO,
            (task_info_t) &info, &infoCount
        ) == KERN_SUCCESS)
        return static_cast<std::uint32_t>(info.resident_size);
    return 0;
}
