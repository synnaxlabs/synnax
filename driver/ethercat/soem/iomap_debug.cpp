/*
 * EtherCAT IOmap Debug Tool
 *
 * Shows the actual IOmap layout as configured by SOEM's ecx_config_map_group().
 * Useful for understanding where each slave's data is located in the IOmap.
 *
 * Build: bazel build //driver/ethercat/soem:iomap_debug
 * Run:   sudo bazel-bin/driver/ethercat/soem/iomap_debug en7
 */

#include <chrono>
#include <csignal>
#include <cstdio>
#include <cstring>
#include <thread>

extern "C" {
#include "soem/soem.h"
}

static volatile bool g_running = true;

void signal_handler(int sig) {
    (void)sig;
    g_running = false;
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        printf("Usage: %s <interface>\n", argv[0]);
        return 1;
    }

    signal(SIGINT, signal_handler);

    const char* ifname = argv[1];
    ecx_contextt ctx;
    memset(&ctx, 0, sizeof(ctx));
    char iomap[4096];
    memset(iomap, 0, sizeof(iomap));

    printf("=== EtherCAT IOmap Debug ===\n\n");

    // Initialize
    if (ecx_init(&ctx, ifname) <= 0) {
        printf("ERROR: Failed to initialize\n");
        return 1;
    }

    // Discover slaves
    int slave_count = ecx_config_init(&ctx);
    if (slave_count <= 0) {
        printf("ERROR: No slaves found\n");
        ecx_close(&ctx);
        return 1;
    }
    printf("Found %d slaves\n\n", slave_count);

    // Exclude problematic 6xSTG modules
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].eep_id == 0x000000FC) {
            ctx.slavelist[i].group = 1;
            printf("Slave %d (%s) -> group 1 (excluded)\n", i, ctx.slavelist[i].name);
        } else {
            ctx.slavelist[i].group = 0;
            printf("Slave %d (%s) -> group 0 (active)\n", i, ctx.slavelist[i].name);
        }
    }

    // Map PDOs - this is where SOEM determines the actual layout
    printf("\nMapping PDOs...\n");
    int iomap_size = ecx_config_map_group(&ctx, iomap, 0);
    printf("IOmap size: %d bytes\n", iomap_size);
    printf("Group 0: Obytes=%d, Ibytes=%d\n\n",
           ctx.grouplist[0].Obytes, ctx.grouplist[0].Ibytes);

    // Show the actual IOmap layout for each slave
    printf("=== Slave IOmap Layout ===\n\n");
    for (int i = 1; i <= slave_count; i++) {
        ec_slavet* sl = &ctx.slavelist[i];
        if (sl->group != 0) {
            printf("Slave %d (%s): EXCLUDED (group %d)\n\n", i, sl->name, sl->group);
            continue;
        }

        printf("Slave %d: %s\n", i, sl->name);
        printf("  Product: 0x%08X\n", sl->eep_id);
        printf("  Outputs: %d bytes at IOmap offset %ld\n",
               sl->Obytes, sl->outputs ? (sl->outputs - (uint8_t*)iomap) : -1);
        printf("  Inputs:  %d bytes at IOmap offset %ld\n",
               sl->Ibytes, sl->inputs ? (sl->inputs - (uint8_t*)iomap) : -1);
        printf("\n");
    }

    // Transition to SAFE_OP then OP
    printf("Transitioning to SAFE_OP...\n");
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].group != 0) continue;
        ctx.slavelist[i].state = EC_STATE_SAFE_OP;
        ecx_writestate(&ctx, i);
    }
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].group != 0) continue;
        ecx_statecheck(&ctx, i, EC_STATE_SAFE_OP, 2000000);
    }

    // Start process data exchange
    ecx_send_processdata(&ctx);
    ecx_receive_processdata(&ctx, 1000);

    printf("Transitioning to OPERATIONAL...\n");
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].group != 0) continue;
        ctx.slavelist[i].state = EC_STATE_OPERATIONAL;
        ecx_writestate(&ctx, i);
    }
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].group != 0) continue;
        uint16_t state = ecx_statecheck(&ctx, i, EC_STATE_OPERATIONAL, 2000000);
        printf("  Slave %d: %s\n", i, (state & 0x0F) == EC_STATE_OPERATIONAL ? "OP" : "FAILED");
    }

    // Run cyclic exchange with output test pattern
    printf("\n=== Cyclic Exchange with Walking Bit ===\n");
    printf("Writing to slave outputs, reading inputs...\n\n");

    uint32_t pattern = 1;
    int cycle = 0;

    // First, show detailed input structure for one 32xDO module
    printf("\n=== Detailed Input Data Structure (Slave 4) ===\n");
    ecx_send_processdata(&ctx);
    ecx_receive_processdata(&ctx, 1000);

    ec_slavet* sl4 = &ctx.slavelist[4];
    if (sl4->inputs && sl4->Ibytes > 0) {
        printf("Slave 4 input data (%d bytes):\n", sl4->Ibytes);
        for (int i = 0; i < sl4->Ibytes; i += 4) {
            uint32_t val = 0;
            int bytes_to_read = (sl4->Ibytes - i < 4) ? (sl4->Ibytes - i) : 4;
            memcpy(&val, sl4->inputs + i, bytes_to_read);
            printf("  Offset %2d: 0x%08X", i, val);
            // Try to interpret as different types
            int16_t* as_int16 = (int16_t*)(sl4->inputs + i);
            printf("  (int16: %d, %d)\n", as_int16[0], (bytes_to_read > 2) ? as_int16[1] : 0);
        }
    }

    printf("\n=== Cyclic Exchange with Walking Bit ===\n");
    printf("Writing to outputs, checking if any input bytes change...\n\n");

    while (g_running && cycle < 50) {
        // Write pattern to each slave's output area
        for (int i = 1; i <= slave_count; i++) {
            ec_slavet* sl = &ctx.slavelist[i];
            if (sl->group != 0 || sl->outputs == nullptr || sl->Obytes < 4) continue;
            memcpy(sl->outputs, &pattern, sizeof(pattern));
        }

        // Exchange process data
        ecx_send_processdata(&ctx);
        int wkc = ecx_receive_processdata(&ctx, 1000);

        cycle++;
        if (cycle % 5 == 0) {
            printf("Cycle %3d | WKC=%d | Out=0x%08X\n", cycle, wkc, pattern);

            // Show first 16 bytes of input for slave 4
            if (sl4->inputs && sl4->Ibytes >= 16) {
                printf("  S4 input[0-15]: ");
                for (int i = 0; i < 16; i++) {
                    printf("%02X ", sl4->inputs[i]);
                }
                printf("\n");
            }

            // Rotate pattern
            pattern = (pattern << 1) | (pattern >> 31);
        }

        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }

    // Cleanup
    printf("\nReturning to INIT...\n");
    ctx.slavelist[0].state = EC_STATE_INIT;
    ecx_writestate(&ctx, 0);
    ecx_close(&ctx);

    printf("Done.\n");
    return 0;
}
