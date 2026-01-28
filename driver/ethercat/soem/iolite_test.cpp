/*
 * IOLITE R8 integration test.
 * Tests the full EtherCAT flow: initialize, activate, read data.
 *
 * Build: bazel build //driver/ethercat/soem:iolite_test
 * Run:   sudo bazel-bin/driver/ethercat/soem/iolite_test en7
 */

#include <chrono>
#include <cstdio>
#include <csignal>
#include <thread>

#include "driver/ethercat/soem/master.h"

using namespace ethercat;
using namespace ethercat::soem;

// Signal handler for graceful shutdown
static volatile bool running = true;

void signal_handler(int sig) {
    (void)sig;
    running = false;
    printf("\nShutdown requested...\n");
}

void print_slaves(const std::vector<SlaveInfo>& slaves) {
    printf("\n=== Discovered Slaves ===\n");
    printf("%-5s %-30s %-12s %-12s\n", "Pos", "Name", "Vendor", "Product");
    printf("%-5s %-30s %-12s %-12s\n", "---", "----", "------", "-------");

    for (const auto& slave : slaves) {
        printf("%-5d %-30s 0x%08X 0x%08X\n",
               slave.position,
               slave.name.c_str(),
               slave.vendor_id,
               slave.product_code);
    }
    printf("\n");
}

const char* state_to_string(SlaveState state) {
    switch (state) {
        case SlaveState::INIT: return "INIT";
        case SlaveState::PRE_OP: return "PRE_OP";
        case SlaveState::BOOT: return "BOOT";
        case SlaveState::SAFE_OP: return "SAFE_OP";
        case SlaveState::OP: return "OP";
        default: return "UNKNOWN";
    }
}

void print_slave_states(SOEMMaster& master, const std::vector<SlaveInfo>& slaves) {
    printf("=== Slave States ===\n");
    for (const auto& slave : slaves) {
        SlaveState state = master.slave_state(slave.position);
        printf("  Slave %d (%s): %s\n",
               slave.position,
               slave.name.c_str(),
               state_to_string(state));
    }
    printf("\n");
}

void dump_hex(const uint8_t* data, size_t len, size_t max_bytes = 64) {
    size_t display_len = (len > max_bytes) ? max_bytes : len;
    for (size_t i = 0; i < display_len; i++) {
        printf("%02X ", data[i]);
        if ((i + 1) % 16 == 0) printf("\n");
    }
    if (display_len % 16 != 0) printf("\n");
    if (len > max_bytes) {
        printf("  ... (%zu more bytes)\n", len - max_bytes);
    }
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        printf("Usage: %s <interface>\n", argv[0]);
        printf("Example: %s en7\n", argv[0]);
        return 1;
    }

    const char* ifname = argv[1];

    // Set up signal handler
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    printf("=== IOLITE R8 Integration Test ===\n");
    printf("Interface: %s\n\n", ifname);

    // Create the SOEM master
    SOEMMaster master(ifname);

    // Step 1: Initialize (discover slaves)
    printf("[1/3] Initializing master...\n");
    auto err = master.initialize();
    if (err) {
        printf("ERROR: Failed to initialize: %s\n", err.message().c_str());
        return 1;
    }
    printf("      Master initialized successfully.\n");

    // Print discovered slaves
    auto slaves = master.slaves();
    print_slaves(slaves);
    print_slave_states(master, slaves);

    // Step 2: Activate (transition to OPERATIONAL)
    printf("[2/3] Activating master (INIT -> PRE_OP -> SAFE_OP -> OP)...\n");
    err = master.activate();
    if (err) {
        printf("ERROR: Failed to activate: %s\n", err.message().c_str());
        return 1;
    }
    printf("      Master activated successfully.\n\n");

    // Print slave states after activation
    print_slave_states(master, slaves);

    // Check if all slaves are operational
    if (master.all_slaves_operational()) {
        printf("All slaves are OPERATIONAL!\n\n");
    } else {
        printf("WARNING: Not all slaves reached OPERATIONAL state.\n\n");
    }

    // Step 3: Run cyclic data exchange
    printf("[3/3] Starting cyclic data exchange (Ctrl+C to stop)...\n\n");

    int cycle_count = 0;
    int error_count = 0;
    auto start_time = std::chrono::steady_clock::now();

    while (running) {
        // Send process data (outputs)
        err = master.send();
        if (err) {
            error_count++;
            if (error_count <= 3) {
                printf("Send error: %s\n", err.message().c_str());
            }
        }

        // Wait a bit for slaves to process
        std::this_thread::sleep_for(std::chrono::milliseconds(1));

        // Receive process data (inputs)
        err = master.receive();
        if (err) {
            error_count++;
            if (error_count <= 3) {
                printf("Receive error: %s\n", err.message().c_str());
            }
        }

        cycle_count++;

        // Print status every second
        auto now = std::chrono::steady_clock::now();
        auto elapsed = std::chrono::duration_cast<std::chrono::seconds>(now - start_time).count();

        if (cycle_count % 100 == 0) {
            printf("\rCycle %d | Errors: %d | Running: %llds    ",
                   cycle_count, error_count, (long long)elapsed);
            fflush(stdout);
        }

        // Sleep to maintain ~100Hz cycle rate
        std::this_thread::sleep_for(std::chrono::milliseconds(10));

        // Limit test duration for safety
        if (elapsed >= 30) {
            printf("\n\nTest duration limit reached (30 seconds).\n");
            break;
        }
    }

    printf("\n\n=== Test Complete ===\n");
    printf("Total cycles: %d\n", cycle_count);
    printf("Total errors: %d\n", error_count);

    // Deactivate and cleanup (handled by destructor, but let's be explicit)
    printf("\nDeactivating master...\n");
    master.deactivate();
    printf("Done.\n");

    return (error_count > 0) ? 1 : 0;
}
