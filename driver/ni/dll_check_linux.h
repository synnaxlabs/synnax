#include <iostream>
#include "glog/logging.h"

bool does_dll_exist(const char* dll_path) {
    LOG(ERROR) << "[ni] " << dll_path << " not found";
    return false;
}
