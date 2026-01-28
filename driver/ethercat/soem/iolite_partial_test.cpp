/*
 * IOLITE R8 partial test - runs with working slaves only.
 * Skips problematic 6xSTG modules to prove the concept works.
 */

#include <cstdio>
#include <cstring>
#include <chrono>
#include <thread>
#include <csignal>

extern "C" {
#include "soem/soem.h"
}

static volatile bool running = true;

void signal_handler(int sig) {
    (void)sig;
    running = false;
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

    printf("=== IOLITE R8 Partial Test (working slaves only) ===\n\n");

    // Initialize
    printf("[1] Initializing...\n");
    if (ecx_init(&ctx, ifname) <= 0) {
        printf("ERROR: Failed to initialize\n");
        return 1;
    }

    // Discover
    printf("[2] Discovering slaves...\n");
    int slave_count = ecx_config_init(&ctx);
    if (slave_count <= 0) {
        printf("ERROR: No slaves found\n");
        ecx_close(&ctx);
        return 1;
    }
    printf("    Found %d slaves\n", slave_count);

    // Put 6xSTG modules in a separate group (group 1) that we won't activate
    printf("[3] Separating problematic slaves into group 1...\n");
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].eep_id == 0x000000FC) {  // 6xSTG
            printf("    Slave %d (%s) -> group 1 (skip)\n", i, ctx.slavelist[i].name);
            ctx.slavelist[i].group = 1;
        } else {
            printf("    Slave %d (%s) -> group 0 (active)\n", i, ctx.slavelist[i].name);
            ctx.slavelist[i].group = 0;
        }
    }

    // Map only group 0
    printf("[4] Mapping PDOs for group 0 only...\n");
    int iomap_size = ecx_config_map_group(&ctx, iomap, 0);
    printf("    IOmap size: %d bytes\n", iomap_size);
    printf("    Group 0: Obytes=%d, Ibytes=%d\n",
           ctx.grouplist[0].Obytes, ctx.grouplist[0].Ibytes);

    // Transition group 0 to SAFE_OP
    printf("[5] Transitioning group 0 to SAFE_OP...\n");

    // First get all to PRE_OP
    ctx.slavelist[0].state = EC_STATE_PRE_OP;
    ecx_writestate(&ctx, 0);
    ecx_statecheck(&ctx, 0, EC_STATE_PRE_OP, 3000000);

    // Now SAFE_OP for group 0 only - need to do it per slave
    int safe_op_count = 0;
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].group == 0) {
            ctx.slavelist[i].state = EC_STATE_SAFE_OP;
            ecx_writestate(&ctx, i);
            uint16_t state = ecx_statecheck(&ctx, i, EC_STATE_SAFE_OP, 2000000);
            if ((state & 0x0F) == EC_STATE_SAFE_OP) {
                printf("    Slave %d: SAFE_OP OK\n", i);
                safe_op_count++;
            } else {
                printf("    Slave %d: FAILED (state=0x%02X, AL=%d)\n",
                       i, state, ctx.slavelist[i].ALstatuscode);
            }
        }
    }
    printf("    %d/%d group 0 slaves in SAFE_OP\n", safe_op_count, slave_count - 2);

    if (safe_op_count == 0) {
        printf("ERROR: No slaves reached SAFE_OP\n");
        ecx_close(&ctx);
        return 1;
    }

    // Exchange process data before OP
    printf("[6] Starting process data exchange...\n");
    ecx_send_processdata(&ctx);
    ecx_receive_processdata(&ctx, 1000);

    // Transition to OPERATIONAL
    printf("[7] Transitioning to OPERATIONAL...\n");
    int op_count = 0;
    for (int i = 1; i <= slave_count; i++) {
        if (ctx.slavelist[i].group == 0 &&
            (ctx.slavelist[i].state & 0x0F) == EC_STATE_SAFE_OP) {
            ctx.slavelist[i].state = EC_STATE_OPERATIONAL;
            ecx_writestate(&ctx, i);
            uint16_t state = ecx_statecheck(&ctx, i, EC_STATE_OPERATIONAL, 2000000);
            if ((state & 0x0F) == EC_STATE_OPERATIONAL) {
                printf("    Slave %d (%s): OPERATIONAL!\n", i, ctx.slavelist[i].name);
                op_count++;
            } else {
                printf("    Slave %d: FAILED (state=0x%02X)\n", i, state);
            }
        }
    }
    printf("    %d slaves OPERATIONAL\n\n", op_count);

    if (op_count == 0) {
        printf("ERROR: No slaves reached OPERATIONAL\n");
        ecx_close(&ctx);
        return 1;
    }

    // Calculate expected WKC for group 0
    int expected_wkc = (ctx.grouplist[0].outputsWKC * 2) + ctx.grouplist[0].inputsWKC;
    printf("Expected WKC: %d\n", expected_wkc);

    // Run cyclic exchange
    printf("[8] Running cyclic data exchange (Ctrl+C to stop)...\n\n");

    int cycle = 0;
    int errors = 0;

    while (running && cycle < 300) {  // Max 30 seconds at 10Hz
        ecx_send_processdata(&ctx);
        int wkc = ecx_receive_processdata(&ctx, 1000);

        cycle++;

        if (cycle % 10 == 0) {
            printf("Cycle %d: WKC=%d", cycle, wkc);

            // Print some input data
            if (ctx.grouplist[0].Ibytes > 0) {
                printf(" | Input data: ");
                uint8_t* inputs = (uint8_t*)ctx.grouplist[0].inputs;
                for (int i = 0; i < 16 && i < (int)ctx.grouplist[0].Ibytes; i++) {
                    printf("%02X ", inputs[i]);
                }
            }
            printf("\n");
        }

        if (wkc < expected_wkc) {
            errors++;
        }

        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }

    printf("\n=== Summary ===\n");
    printf("Cycles: %d\n", cycle);
    printf("Errors: %d\n", errors);
    printf("Input bytes: %d\n", ctx.grouplist[0].Ibytes);
    printf("Output bytes: %d\n", ctx.grouplist[0].Obytes);

    // Cleanup
    printf("\nReturning to INIT...\n");
    ctx.slavelist[0].state = EC_STATE_INIT;
    ecx_writestate(&ctx, 0);
    ecx_statecheck(&ctx, 0, EC_STATE_INIT, 1000000);

    ecx_close(&ctx);
    printf("Done.\n");

    return errors > 0 ? 1 : 0;
}
