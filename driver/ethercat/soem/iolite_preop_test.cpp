/*
 * IOLITE R8 PRE-OP test - tests state transitions without PDO mapping.
 * This helps isolate whether the issue is PDO mapping or state transitions.
 *
 * Build: bazel build //driver/ethercat/soem:iolite_preop_test
 * Run:   sudo bazel-bin/driver/ethercat/soem/iolite_preop_test en7
 */

#include <cstdio>
#include <cstring>
#include <chrono>
#include <thread>

extern "C" {
#include "soem/soem.h"
}

const char* state_to_string(uint16_t state) {
    switch (state & 0x0F) {
        case EC_STATE_INIT: return "INIT";
        case EC_STATE_PRE_OP: return "PRE_OP";
        case EC_STATE_BOOT: return "BOOT";
        case EC_STATE_SAFE_OP: return "SAFE_OP";
        case EC_STATE_OPERATIONAL: return "OP";
        default: return "UNKNOWN";
    }
}

void print_slave_states(ecx_contextt* ctx) {
    ecx_readstate(ctx);
    printf("\nSlave States:\n");
    for (int i = 1; i <= ctx->slavecount; i++) {
        uint16_t state = ctx->slavelist[i].state;
        printf("  Slave %d (%s): %s",
               i, ctx->slavelist[i].name, state_to_string(state));
        if (state & 0x10) {
            printf(" [ERROR: AL code %d]", ctx->slavelist[i].ALstatuscode);
        }
        printf("\n");
    }
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        printf("Usage: %s <interface>\n", argv[0]);
        return 1;
    }

    const char* ifname = argv[1];
    ecx_contextt ctx;
    memset(&ctx, 0, sizeof(ctx));

    printf("=== IOLITE R8 PRE-OP Test ===\n\n");

    // Initialize
    printf("[1] Initializing on %s...\n", ifname);
    if (ecx_init(&ctx, ifname) <= 0) {
        printf("ERROR: Failed to initialize\n");
        return 1;
    }

    // Discover slaves
    printf("[2] Discovering slaves...\n");
    int slave_count = ecx_config_init(&ctx);
    if (slave_count <= 0) {
        printf("ERROR: No slaves found\n");
        ecx_close(&ctx);
        return 1;
    }
    printf("    Found %d slaves\n", slave_count);
    print_slave_states(&ctx);

    // Transition to PRE_OP (no PDO mapping needed)
    printf("\n[3] Transitioning to PRE_OP...\n");
    ctx.slavelist[0].state = EC_STATE_PRE_OP;
    ecx_writestate(&ctx, 0);

    // Wait for state change
    uint16_t state = ecx_statecheck(&ctx, 0, EC_STATE_PRE_OP, 3000000);
    print_slave_states(&ctx);

    if ((state & 0x0F) != EC_STATE_PRE_OP) {
        printf("ERROR: Failed to reach PRE_OP\n");
    } else {
        printf("SUCCESS: All slaves in PRE_OP\n");
    }

    // Now try to read more detailed PDO info via SDO
    printf("\n[4] Reading PDO configuration via CoE SDO...\n");

    for (int slave = 1; slave <= slave_count; slave++) {
        printf("\n--- Slave %d: %s ---\n", slave, ctx.slavelist[slave].name);

        // Read object 0x1000 - Device Type
        uint32_t device_type = 0;
        int size = sizeof(device_type);
        int wkc = ecx_SDOread(&ctx, slave, 0x1000, 0, FALSE, &size, &device_type, EC_TIMEOUTRXM);
        if (wkc > 0) {
            printf("  Device Type (0x1000): 0x%08X\n", device_type);
        }

        // Read object 0x1018 - Identity
        uint32_t vendor_id = 0;
        size = sizeof(vendor_id);
        wkc = ecx_SDOread(&ctx, slave, 0x1018, 1, FALSE, &size, &vendor_id, EC_TIMEOUTRXM);
        if (wkc > 0) {
            printf("  Vendor ID (0x1018:1): 0x%08X\n", vendor_id);
        }

        // Try to list available PDOs by reading 0x1600-0x17FF (RxPDO mapping)
        // and 0x1A00-0x1BFF (TxPDO mapping)
        printf("  Checking RxPDO objects (0x1600-0x1603):\n");
        for (uint16_t idx = 0x1600; idx <= 0x1603; idx++) {
            uint8_t num_entries = 0;
            size = sizeof(num_entries);
            wkc = ecx_SDOread(&ctx, slave, idx, 0, FALSE, &size, &num_entries, EC_TIMEOUTRXM);
            if (wkc > 0 && num_entries > 0) {
                printf("    0x%04X: %d entries\n", idx, num_entries);
                for (int e = 1; e <= num_entries && e <= 8; e++) {
                    uint32_t entry = 0;
                    size = sizeof(entry);
                    wkc = ecx_SDOread(&ctx, slave, idx, e, FALSE, &size, &entry, EC_TIMEOUTRXM);
                    if (wkc > 0) {
                        uint16_t obj_idx = (entry >> 16) & 0xFFFF;
                        uint8_t obj_sub = (entry >> 8) & 0xFF;
                        uint8_t bit_len = entry & 0xFF;
                        printf("      [%d] 0x%04X:%02X (%d bits)\n", e, obj_idx, obj_sub, bit_len);
                    }
                }
            }
        }

        printf("  Checking TxPDO objects (0x1A00-0x1A03):\n");
        for (uint16_t idx = 0x1A00; idx <= 0x1A03; idx++) {
            uint8_t num_entries = 0;
            size = sizeof(num_entries);
            wkc = ecx_SDOread(&ctx, slave, idx, 0, FALSE, &size, &num_entries, EC_TIMEOUTRXM);
            if (wkc > 0 && num_entries > 0) {
                printf("    0x%04X: %d entries\n", idx, num_entries);
                for (int e = 1; e <= num_entries && e <= 8; e++) {
                    uint32_t entry = 0;
                    size = sizeof(entry);
                    wkc = ecx_SDOread(&ctx, slave, idx, e, FALSE, &size, &entry, EC_TIMEOUTRXM);
                    if (wkc > 0) {
                        uint16_t obj_idx = (entry >> 16) & 0xFFFF;
                        uint8_t obj_sub = (entry >> 8) & 0xFF;
                        uint8_t bit_len = entry & 0xFF;
                        printf("      [%d] 0x%04X:%02X (%d bits)\n", e, obj_idx, obj_sub, bit_len);
                    }
                }
            }
        }
    }

    // Now try PDO mapping with inputs only (skip outputs for problematic slaves)
    printf("\n[5] Attempting PDO mapping (inputs only for STG modules)...\n");

    // Use SOEM's config_map but first disable outputs for the STG slaves
    // by setting their SM2 to 0
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].eep_id == 0x000000FC) {  // 6xSTG
            printf("    Disabling outputs for slave %d (6xSTG)\n", i);
            // Clear SM2 (output SM) configuration
            ctx.slavelist[i].SM[2].StartAddr = 0;
            ctx.slavelist[i].SM[2].SMlength = 0;
            ctx.slavelist[i].SMtype[2] = 0;  // Unused
        }
    }

    // Now try config_map_group
    char iomap[4096];
    memset(iomap, 0, sizeof(iomap));
    int iomap_size = ecx_config_map_group(&ctx, iomap, 0);

    printf("    IOmap size: %d bytes\n", iomap_size);
    printf("    Group 0: Obytes=%d, Ibytes=%d\n",
           ctx.grouplist[0].Obytes, ctx.grouplist[0].Ibytes);

    // Check state after mapping
    print_slave_states(&ctx);

    // Try to go to SAFE_OP
    printf("\n[6] Transitioning to SAFE_OP...\n");
    ctx.slavelist[0].state = EC_STATE_SAFE_OP;
    ecx_writestate(&ctx, 0);
    state = ecx_statecheck(&ctx, 0, EC_STATE_SAFE_OP, 3000000);
    print_slave_states(&ctx);

    if ((state & 0x0F) == EC_STATE_SAFE_OP) {
        printf("SUCCESS: All slaves in SAFE_OP\n");

        // Try OPERATIONAL
        printf("\n[7] Transitioning to OPERATIONAL...\n");

        // Need to exchange process data before OP
        ecx_send_processdata(&ctx);
        ecx_receive_processdata(&ctx, 1000);

        ctx.slavelist[0].state = EC_STATE_OPERATIONAL;
        ecx_writestate(&ctx, 0);
        state = ecx_statecheck(&ctx, 0, EC_STATE_OPERATIONAL, 3000000);
        print_slave_states(&ctx);

        if ((state & 0x0F) == EC_STATE_OPERATIONAL) {
            printf("SUCCESS: All slaves OPERATIONAL!\n");

            // Do a few cycles
            printf("\n[8] Running 10 process data cycles...\n");
            for (int c = 0; c < 10; c++) {
                ecx_send_processdata(&ctx);
                int wkc = ecx_receive_processdata(&ctx, 1000);
                printf("  Cycle %d: WKC=%d\n", c + 1, wkc);
                std::this_thread::sleep_for(std::chrono::milliseconds(100));
            }
        }
    }

    // Cleanup
    printf("\n[9] Returning to INIT and closing...\n");
    ctx.slavelist[0].state = EC_STATE_INIT;
    ecx_writestate(&ctx, 0);
    ecx_statecheck(&ctx, 0, EC_STATE_INIT, 1000000);

    ecx_close(&ctx);
    printf("Done.\n");

    return 0;
}
