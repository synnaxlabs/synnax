/*
 * EtherCAT slave information dump.
 * Shows detailed configuration for each slave including PDO mappings.
 *
 * Build: bazel build //driver/ethercat/soem:slave_info
 * Run:   sudo bazel-bin/driver/ethercat/soem/slave_info en7
 */

#include <cstdio>
#include <cstring>

extern "C" {
#include "soem/soem.h"
}

void print_slave_info(ecx_contextt* ctx, int slave) {
    ec_slavet* sl = &ctx->slavelist[slave];

    printf("\n========================================\n");
    printf("Slave %d: %s\n", slave, sl->name);
    printf("========================================\n");

    printf("Vendor ID:     0x%08X\n", sl->eep_man);
    printf("Product Code:  0x%08X\n", sl->eep_id);
    printf("Revision:      0x%08X\n", sl->eep_rev);
    printf("Serial:        0x%08X\n", sl->eep_ser);

    printf("\nMailbox:\n");
    printf("  Mailbox supported: %s\n", (sl->mbx_l > 0) ? "Yes" : "No");
    if (sl->mbx_l > 0) {
        printf("  Mailbox protocols: 0x%04X\n", sl->mbx_proto);
        printf("    CoE: %s\n", (sl->mbx_proto & ECT_MBXPROT_COE) ? "Yes" : "No");
        printf("    FoE: %s\n", (sl->mbx_proto & ECT_MBXPROT_FOE) ? "Yes" : "No");
        printf("    SoE: %s\n", (sl->mbx_proto & ECT_MBXPROT_SOE) ? "Yes" : "No");
        printf("    EoE: %s\n", (sl->mbx_proto & ECT_MBXPROT_EOE) ? "Yes" : "No");
    }

    printf("\nSyncManager Configuration:\n");
    for (int i = 0; i < EC_MAXSM; i++) {
        if (sl->SM[i].StartAddr != 0 || sl->SMtype[i] != 0) {
            const char* type_str;
            switch (sl->SMtype[i]) {
                case 0: type_str = "Unused"; break;
                case 1: type_str = "Mailbox Out (master->slave)"; break;
                case 2: type_str = "Mailbox In (slave->master)"; break;
                case 3: type_str = "Process Data Out (outputs)"; break;
                case 4: type_str = "Process Data In (inputs)"; break;
                default: type_str = "Unknown"; break;
            }
            printf("  SM%d: Start=0x%04X, Length=%d, Type=%d (%s)\n",
                   i, sl->SM[i].StartAddr, sl->SM[i].SMlength,
                   sl->SMtype[i], type_str);
        }
    }

    printf("\nFMMU Configuration:\n");
    for (int i = 0; i < EC_MAXFMMU; i++) {
        if (sl->FMMU[i].LogStart != 0 || sl->FMMU[i].LogLength != 0) {
            printf("  FMMU%d: LogStart=0x%08X, LogLen=%d, PhysStart=0x%04X, Type=%d\n",
                   i, sl->FMMU[i].LogStart, sl->FMMU[i].LogLength,
                   sl->FMMU[i].PhysStart, sl->FMMU[i].FMMUtype);
        }
    }

    printf("\nProcess Data:\n");
    printf("  Output bits:  %d (%d bytes)\n", sl->Obits, sl->Obytes);
    printf("  Input bits:   %d (%d bytes)\n", sl->Ibits, sl->Ibytes);

    if (sl->Obytes > 0) {
        printf("  Output offset: %d (startbit: %d)\n", sl->Ooffset, sl->Ostartbit);
    }
    if (sl->Ibytes > 0) {
        printf("  Input offset:  %d (startbit: %d)\n", sl->Ioffset, sl->Istartbit);
    }

    printf("\nState: 0x%02X", sl->state);
    if (sl->state & 0x10) printf(" (ERROR)");
    printf("\n");
    if (sl->ALstatuscode != 0) {
        printf("AL Status Code: %d (0x%04X)\n", sl->ALstatuscode, sl->ALstatuscode);
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

    printf("Initializing EtherCAT on %s...\n", ifname);

    if (ecx_init(&ctx, ifname) <= 0) {
        printf("ERROR: Failed to initialize interface\n");
        return 1;
    }

    printf("Scanning for slaves...\n");
    int slave_count = ecx_config_init(&ctx);

    if (slave_count <= 0) {
        printf("No slaves found.\n");
        ecx_close(&ctx);
        return 1;
    }

    printf("Found %d slave(s).\n", slave_count);

    // Print basic info for each slave (before mapping)
    for (int i = 1; i <= slave_count; i++) {
        print_slave_info(&ctx, i);
    }

    // Try to read PDO configuration via CoE (SDO)
    printf("\n\n========================================\n");
    printf("Attempting PDO mapping read via CoE...\n");
    printf("========================================\n");

    for (int slave = 1; slave <= slave_count; slave++) {
        ec_slavet* sl = &ctx.slavelist[slave];

        if (!(sl->mbx_proto & ECT_MBXPROT_COE)) {
            printf("\nSlave %d: No CoE support\n", slave);
            continue;
        }

        printf("\nSlave %d (%s):\n", slave, sl->name);

        // Read 0x1C00 - SM communication type
        uint8_t sm_types[8];
        int size = sizeof(sm_types);
        int wkc = ecx_SDOread(&ctx, slave, 0x1C00, 0, FALSE, &size, sm_types, EC_TIMEOUTRXM);
        if (wkc > 0) {
            printf("  SM Comm Types (0x1C00): ");
            for (int i = 0; i < size && i < 8; i++) {
                printf("SM%d=%d ", i, sm_types[i]);
            }
            printf("\n");
        }

        // Read 0x1C12 - RxPDO assign (outputs)
        uint16_t rxpdo_assign[16];
        size = sizeof(rxpdo_assign);
        wkc = ecx_SDOread(&ctx, slave, 0x1C12, 0, FALSE, &size, rxpdo_assign, EC_TIMEOUTRXM);
        if (wkc > 0 && size >= 2) {
            int count = rxpdo_assign[0];
            printf("  RxPDO assign (0x1C12): %d entries\n", count);
            for (int i = 0; i < count && i < 8; i++) {
                printf("    [%d] 0x%04X\n", i, rxpdo_assign[i + 1]);
            }
        } else {
            printf("  RxPDO assign (0x1C12): not available or empty\n");
        }

        // Read 0x1C13 - TxPDO assign (inputs)
        uint16_t txpdo_assign[16];
        size = sizeof(txpdo_assign);
        wkc = ecx_SDOread(&ctx, slave, 0x1C13, 0, FALSE, &size, txpdo_assign, EC_TIMEOUTRXM);
        if (wkc > 0 && size >= 2) {
            int count = txpdo_assign[0];
            printf("  TxPDO assign (0x1C13): %d entries\n", count);
            for (int i = 0; i < count && i < 8; i++) {
                printf("    [%d] 0x%04X\n", i, txpdo_assign[i + 1]);
            }
        } else {
            printf("  TxPDO assign (0x1C13): not available or empty\n");
        }
    }

    ecx_close(&ctx);
    printf("\nDone.\n");
    return 0;
}
