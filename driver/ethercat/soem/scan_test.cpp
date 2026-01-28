/*
 * Simple EtherCAT slave detection test using SOEM (debug version).
 */

#include <cstdio>
#include <cstring>

extern "C" {
#include "soem/soem.h"
}

int main(int argc, char *argv[]) {
    if (argc < 2) {
        printf("Usage: %s <interface>\n", argv[0]);
        printf("\nAvailable interfaces:\n");

        ec_adaptert *adapter = ec_find_adapters();
        while (adapter != nullptr) {
            printf("  %s - %s\n", adapter->name, adapter->desc);
            adapter = adapter->next;
        }
        return 1;
    }

    const char *ifname = argv[1];
    printf("Scanning for EtherCAT slaves on interface: %s\n", ifname);

    ecx_contextt context;
    memset(&context, 0, sizeof(context));

    printf("DEBUG: Calling ecx_init()...\n");
    int init_result = ecx_init(&context, ifname);
    printf("DEBUG: ecx_init() returned %d\n", init_result);

    if (init_result <= 0) {
        printf("ERROR: Failed to initialize interface %s\n", ifname);
        return 1;
    }

    printf("DEBUG: Calling ecx_config_init()...\n");
    int slave_count = ecx_config_init(&context);
    printf("DEBUG: ecx_config_init() returned %d\n", slave_count);
    printf("DEBUG: context.slavecount = %d\n", context.slavecount);

    // Check slavelist even if config_init returned 0
    printf("\nDEBUG: Checking slavelist directly:\n");
    for (int i = 0; i <= 10; i++) {
        if (context.slavelist[i].eep_man != 0 ||
            context.slavelist[i].eep_id != 0 ||
            context.slavelist[i].state != 0) {
            printf("  Slot %d: man=0x%08X id=0x%08X state=%d name='%s'\n",
                   i,
                   context.slavelist[i].eep_man,
                   context.slavelist[i].eep_id,
                   context.slavelist[i].state,
                   context.slavelist[i].name);
        }
    }

    if (slave_count <= 0 && context.slavecount <= 0) {
        printf("\nNo slaves detected.\n");

        // Let's manually try a BRD (Broadcast Read) to see if anything responds
        printf("\nDEBUG: Trying manual broadcast read...\n");

        // Read ALStatus register (0x0130) from all slaves
        int wkc;
        uint8 buf[64];
        memset(buf, 0, sizeof(buf));

        wkc = ecx_BRD(&context.port, 0x0000, 0x0130, sizeof(uint16), buf, EC_TIMEOUTRET);
        printf("DEBUG: BRD to ALStatus returned WKC=%d, data=0x%02X%02X\n",
               wkc, buf[1], buf[0]);

        if (wkc > 0) {
            printf("\n*** Slaves ARE responding (WKC=%d) but config_init failed! ***\n", wkc);
            printf("This suggests a timing or receive issue in the driver.\n");
        }

        ecx_close(&context);
        return 1;
    }

    printf("\nFound %d slave(s):\n", slave_count);
    printf("%-5s %-32s %-10s %-10s %s\n",
           "Pos", "Name", "Vendor", "Product", "State");
    printf("%-5s %-32s %-10s %-10s %s\n",
           "---", "----", "------", "-------", "-----");

    for (int i = 1; i <= slave_count; i++) {
        const char *state_str;
        switch (context.slavelist[i].state) {
            case EC_STATE_INIT:    state_str = "INIT"; break;
            case EC_STATE_PRE_OP:  state_str = "PRE_OP"; break;
            case EC_STATE_SAFE_OP: state_str = "SAFE_OP"; break;
            case EC_STATE_OPERATIONAL: state_str = "OP"; break;
            default: state_str = "UNKNOWN"; break;
        }

        printf("%-5d %-32s 0x%08X 0x%08X %s\n",
               i,
               context.slavelist[i].name,
               context.slavelist[i].eep_man,
               context.slavelist[i].eep_id,
               state_str);
    }

    ecx_close(&context);
    printf("\nScan complete.\n");
    return 0;
}
