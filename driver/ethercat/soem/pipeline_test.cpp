/*
 * EtherCAT Pipeline Integration Test
 *
 * This test exercises the full driver pipeline:
 *   SOEMMaster → CyclicEngine → ReadTaskSource + WriteTaskSink
 *
 * It proves that the core components work end-to-end with real hardware,
 * including both reading inputs and writing outputs.
 *
 * Build: bazel build //driver/ethercat/soem:pipeline_test
 * Run:   sudo bazel-bin/driver/ethercat/soem/pipeline_test en7
 */

#include <atomic>
#include <chrono>
#include <csignal>
#include <cstdio>
#include <cstring>
#include <thread>
#include <vector>

#include "driver/ethercat/cyclic_engine.h"
#include "driver/ethercat/soem/master.h"

using namespace ethercat;
using namespace ethercat::soem;

static volatile bool g_running = true;

void signal_handler(int sig) {
    (void)sig;
    g_running = false;
}

void print_slaves(const std::vector<SlaveInfo>& slaves) {
    printf("\n=== Discovered Slaves ===\n");
    for (const auto& s : slaves) {
        printf("  [%d] %s (Vendor: 0x%08X, Product: 0x%08X)\n",
               s.position, s.name.c_str(), s.vendor_id, s.product_code);
    }
    printf("\n");
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        printf("Usage: %s <interface>\n", argv[0]);
        printf("Example: %s en7\n", argv[0]);
        return 1;
    }

    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    const char* ifname = argv[1];

    printf("=== EtherCAT Pipeline Integration Test ===\n");
    printf("Interface: %s\n", ifname);

    // =========================================================================
    // Step 1: Create the SOEM Master
    // =========================================================================
    printf("\n[1] Creating SOEMMaster...\n");
    auto master = std::make_shared<SOEMMaster>(ifname);

    printf("[2] Initializing master (discovering slaves)...\n");
    auto err = master->initialize();
    if (err) {
        printf("ERROR: Master initialization failed: %s\n", err.message().c_str());
        return 1;
    }

    auto slaves = master->slaves();
    print_slaves(slaves);

    if (slaves.empty()) {
        printf("ERROR: No slaves found\n");
        return 1;
    }

    // =========================================================================
    // Step 2: Create the CyclicEngine
    // =========================================================================
    printf("[3] Creating CyclicEngine (10ms cycle time)...\n");
    CyclicEngineConfig engine_config(telem::MILLISECOND * 10);
    auto engine = std::make_shared<CyclicEngine>(master, engine_config);

    // =========================================================================
    // Step 3: Register PDOs for reading and writing
    // =========================================================================
    printf("[4] Registering PDOs...\n");

    // We'll register both input and output PDOs for the 32xDO BUS2 modules
    // Input (TxPDO): 0x6000 region - status/feedback of digital outputs
    // Output (RxPDO): 0x7000 region - control of digital outputs

    struct PDOInfo {
        uint16_t slave_position;
        size_t input_offset;
        size_t output_offset;
        std::string name;
    };
    std::vector<PDOInfo> registered_pdos;

    for (const auto& slave : slaves) {
        // Only register PDOs for the 32xDO modules (product code 0xFB)
        if (slave.product_code != 0x000000FB) {
            printf("    Skipping slave %d (%s) - not a 32xDO module\n",
                   slave.position, slave.name.c_str());
            continue;
        }

        // Register a 32-bit input PDO: 0x6000:06 (output status feedback)
        PDOEntry input_entry;
        input_entry.slave_position = slave.position;
        input_entry.index = 0x6000;
        input_entry.subindex = 6;
        input_entry.bit_length = 32;
        input_entry.is_input = true;

        auto [input_offset, input_err] = engine->register_input_pdo(input_entry);
        if (input_err) {
            printf("    ERROR registering input PDO for slave %d: %s\n",
                   slave.position, input_err.message().c_str());
            continue;
        }

        // Register a 32-bit output PDO: 0x7000:06 (digital output control)
        PDOEntry output_entry;
        output_entry.slave_position = slave.position;
        output_entry.index = 0x7000;
        output_entry.subindex = 6;
        output_entry.bit_length = 32;
        output_entry.is_input = false;

        auto [output_offset, output_err] = engine->register_output_pdo(output_entry);
        if (output_err) {
            printf("    ERROR registering output PDO for slave %d: %s\n",
                   slave.position, output_err.message().c_str());
            continue;
        }

        printf("    Slave %d: input@%zu, output@%zu\n",
               slave.position, input_offset, output_offset);
        registered_pdos.push_back({slave.position, input_offset, output_offset, slave.name});
    }

    if (registered_pdos.empty()) {
        printf("WARNING: No PDOs registered. Continuing anyway to test cyclic engine.\n");
    }

    // =========================================================================
    // Step 4: Start the CyclicEngine (adds a "task")
    // =========================================================================
    printf("\n[5] Starting CyclicEngine (activating master)...\n");
    err = engine->add_task();
    if (err) {
        printf("ERROR: Failed to start engine: %s\n", err.message().c_str());
        return 1;
    }

    printf("    CyclicEngine running: %s\n", engine->running() ? "YES" : "NO");
    printf("    Cycle time: %.0f us\n", engine->cycle_time().microseconds());
    printf("    Task count: %d\n", engine->task_count());

    // =========================================================================
    // Step 5: Read and Write data (simulating ReadTaskSource + WriteTaskSink)
    // =========================================================================
    printf("\n[6] Reading/Writing process data (Ctrl+C to stop)...\n");
    printf("    Writing walking bit pattern to outputs, reading feedback...\n\n");

    std::vector<uint8_t> input_buffer(256);
    std::atomic<bool> stopped{false};

    int read_count = 0;
    int error_count = 0;
    uint32_t output_pattern = 1;  // Start with bit 0 set
    auto start_time = std::chrono::steady_clock::now();

    while (g_running && read_count < 100) {
        // Write output pattern to all 32xDO modules (simulating WriteTaskSink)
        for (const auto& pdo : registered_pdos) {
            engine->write_output(pdo.output_offset, &output_pattern, sizeof(output_pattern));
        }

        // Wait for new input data (this is what ReadTaskSource::read() does)
        auto wait_err = engine->wait_for_inputs(input_buffer, stopped);

        if (wait_err) {
            error_count++;
            if (error_count <= 3) {
                printf("Read error: %s\n", wait_err.message().c_str());
            }
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
            continue;
        }

        read_count++;

        // Print data every 10 reads
        if (read_count % 10 == 0) {
            printf("Read %3d | Out=0x%08X | ", read_count, output_pattern);

            // Show input feedback from each slave
            for (const auto& pdo : registered_pdos) {
                if (pdo.input_offset + 4 <= input_buffer.size()) {
                    uint32_t value;
                    memcpy(&value, input_buffer.data() + pdo.input_offset, sizeof(value));
                    printf("S%d=0x%08X ", pdo.slave_position, value);
                }
            }

            // Show match status
            bool all_match = true;
            for (const auto& pdo : registered_pdos) {
                if (pdo.input_offset + 4 <= input_buffer.size()) {
                    uint32_t value;
                    memcpy(&value, input_buffer.data() + pdo.input_offset, sizeof(value));
                    if (value != output_pattern) all_match = false;
                }
            }
            printf("| %s\n", all_match ? "MATCH" : "no match");
        }

        // Rotate the walking bit pattern every 10 cycles
        if (read_count % 10 == 0) {
            output_pattern = (output_pattern << 1) | (output_pattern >> 31);  // Rotate left
        }
    }

    auto end_time = std::chrono::steady_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(
        end_time - start_time).count();

    // =========================================================================
    // Step 6: Stop and cleanup
    // =========================================================================
    printf("\n[7] Stopping CyclicEngine...\n");
    engine->remove_task();

    printf("\n=== Test Summary ===\n");
    printf("Total reads: %d\n", read_count);
    printf("Errors: %d\n", error_count);
    printf("Duration: %lld ms\n", (long long)duration);
    printf("Effective rate: %.1f Hz\n",
           read_count > 0 ? (read_count * 1000.0 / duration) : 0);
    printf("Final cycle count: %llu\n", (unsigned long long)engine->cycle_count());

    auto last_err = engine->last_error();
    if (last_err) {
        printf("Last engine error: %s\n", last_err.message().c_str());
    }

    printf("\n=== Pipeline Test Complete ===\n");

    return (error_count > read_count / 2) ? 1 : 0;
}
