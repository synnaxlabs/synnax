#include <iostream>
#include <windows.h>
#include "glog/logging.h"

bool does_dll_exist(const char* dll_path) {
    HMODULE hModule = LoadLibrary(dll_path);
    if (hModule == NULL) {
        LOG(ERROR) << "[ni] " << dll_path << " not found";
        return false;
    }
    FreeLibrary(hModule);
    return true;
}