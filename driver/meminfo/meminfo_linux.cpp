#include "meminfo.h"
#include <iostream>
#include <fstream>
#include <string>

std::uint32_t meminfo::getUsage() {
    std::ifstream status("/proc/self/status", std::ios_base::in);
    std::string line;
    std::string key;
    std::uint32_t memory = 0;

    while (std::getline(status, line)) {
        if (line.substr(0, 6) == "VmRSS:") {
            std::istringstream iss(line);
            iss >> key >> memory; // "VmRSS" and the memory value
            return memory * 1024; // Convert from kB to Bytes
        }
    }

    std::cerr << "Failed to read memory usage information.\n";
    return 0;
}
