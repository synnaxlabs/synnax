# EtherCAT Testing Notes

Notes from hands-on testing with DEWESoft IOLITE R8 hardware.

## Hardware Setup

### DEWESoft IOLITE R8

Modular DAQ system with internal EtherCAT bus:
- **Connection**: RJ45 port (labeled "EtherCAT", not TCP/IP)
- **Interface**: Belkin USB-C to Ethernet adapter → macOS interface `en7`

### Discovered Slaves

| Position | Name | Vendor ID | Product Code | Status |
|----------|------|-----------|--------------|--------|
| 1 | IOLiteGATE_LATCH2.00 | 0xDEBE50F7 | 0x000000FA | ✅ Works |
| 2 | IOLITEr 6xSTG | 0xDEBE50F7 | 0x000000FC | ❌ Fails |
| 3 | IOLITEr 6xSTG | 0xDEBE50F7 | 0x000000FC | ❌ Fails |
| 4 | IOLITEr 32xDO BUS2 | 0xDEBE50F7 | 0x000000FB | ✅ Works |
| 5 | IOLITEr 32xDO BUS2 | 0xDEBE50F7 | 0x000000FB | ✅ Works |
| 6 | IOLITEr 32xDO BUS2 | 0xDEBE50F7 | 0x000000FB | ✅ Works |
| 7 | IOLITEr 32xDO BUS2 | 0xDEBE50F7 | 0x000000FB | ✅ Works |

## macOS SOEM Issues

### TX Echo Problem

**Symptom**: Slaves detected but always showing 0 slaves or immediate timeout.

**Root cause**: macOS pcap receives TX frames echoed back. SOEM was processing
its own transmitted frame as the response.

**Fix**: Added filter in `nicdrv.c` to skip frames with our exact source MAC:
```c
/* On macOS, pcap sees our own TX frames echoed back.
 * Skip frames that have our exact source MAC - these are TX echoes.
 * Real responses from EtherCAT slaves will have a modified MAC. */
if ((ehp->sa0 == htons(priMAC[0])) &&
    (ehp->sa1 == htons(priMAC[1])) &&
    (ehp->sa2 == htons(priMAC[2])))
{
   pthread_mutex_unlock(&(port->rx_mutex));
   return EC_OTHERFRAME;
}
```

**Key insight**: EtherCAT slaves modify the source MAC slightly (e.g., first
byte 0x00 → 0x02), allowing us to distinguish TX echoes from real responses.

## 6xSTG Module Failure Analysis

### Symptoms

- Fails transition to SAFE_OP
- AL Status Code: 38 (Invalid SM OUT configuration)
- Error flag (0x10) set in state

### Investigation

SDO reads showed the modules have:
- RxPDO (0x1600): Output PDO mapping
- TxPDO (0x1A00): Input PDO mapping

But the SyncManager configuration from EEPROM doesn't match what SOEM configures.

### Workaround

Assign 6xSTG modules to group 1 before `ecx_config_map_group()`:
```c
for (int i = 1; i <= slave_count; i++) {
    if (ctx.slavelist[i].eep_id == 0x000000FC) {  // 6xSTG
        ctx.slavelist[i].group = 1;  // Exclude from group 0
    }
}
ecx_config_map_group(&ctx, iomap, 0);  // Only map group 0
```

### Proper Fix (Future)

Would require:
1. Reading the ESI file for correct SM configuration
2. Manually configuring SyncManagers via SDO before mapping
3. Or, DEWESoft-specific initialization sequence

## IOmap Layout Analysis

### After ecx_config_map_group()

```
Total IOmap: 249 bytes
  Outputs (Obytes): 18 bytes
  Inputs (Ibytes): 224 bytes

Layout: [OUTPUTS 0-17][INPUTS 18-241]
```

### Per-Slave Breakdown

| Slave | Outputs | Output Offset | Inputs | Input Offset |
|-------|---------|---------------|--------|--------------|
| 1 (Gateway) | 2 bytes | 0 | 36 bytes | 18 |
| 4 (32xDO) | 4 bytes | 2 | 30 bytes | 122 |
| 5 (32xDO) | 4 bytes | 6 | 30 bytes | 152 |
| 6 (32xDO) | 4 bytes | 10 | 30 bytes | 182 |
| 7 (32xDO) | 4 bytes | 14 | 30 bytes | 212 |

**Note**: Slaves 2 and 3 (6xSTG) excluded, so no IOmap allocation.

### 32xDO Input Data Structure

Each 32xDO module provides 30 bytes of input:
```
Offset  0-3:  0x33336300 - Status/diagnostic word 1
Offset  4-7:  0x55553333 - Status/diagnostic word 2
Offset  8-11: 0x88885555 - Status/diagnostic word 3
Offset 12-15: 0x00008888 - Status/diagnostic word 4
Offset 16-29: 0x00000000 - Reserved/unused
```

These are NOT output feedback. The values (0x33, 0x55, 0x88) appear to be
diagnostic/status codes, possibly supply voltage or temperature readings.

## Test Results

### Cyclic Exchange Performance

| Metric | Value |
|--------|-------|
| Configured cycle time | 10ms |
| Achieved cycle rate | ~100 Hz |
| Working counter (expected) | 7 |
| Working counter (achieved) | 7 |
| State transitions | All group 0 slaves reached OP |

### Successful Operations

1. ✅ Initialize SOEM on macOS with pcap
2. ✅ Discover all 7 slaves on internal bus
3. ✅ Read slave identification (vendor, product, name)
4. ✅ Exclude problematic slaves via grouping
5. ✅ Map PDOs for group 0
6. ✅ Transition 5 slaves to OPERATIONAL
7. ✅ Cyclic process data exchange
8. ✅ Write to digital outputs
9. ✅ Read input/status data

### Known Issues

1. ❌ 6xSTG modules fail SAFE_OP transition
2. ⚠️ Gateway module sometimes stuck in PRE_OP (doesn't affect operation)
3. ⚠️ Occasional WKC=-1 on first cycle (normal startup behavior)

## Test Programs

Located in `driver/ethercat/soem/`:

| Program | Purpose |
|---------|---------|
| `scan_test` | Basic slave discovery |
| `slave_info` | Detailed slave configuration dump |
| `iolite_preop_test` | PRE_OP state with SDO reads |
| `iolite_partial_test` | Partial operation (exclude 6xSTG) |
| `iolite_test` | Full integration (fails on 6xSTG) |
| `pipeline_test` | CyclicEngine integration test |
| `iomap_debug` | IOmap layout analysis |
| `pcap_debug` | Raw frame debugging |

## Commands Reference

### Build
```bash
bazel build //driver/ethercat/soem:pipeline_test
```

### Run (requires root for raw sockets)
```bash
sudo bazel-bin/driver/ethercat/soem/pipeline_test en7
```

### Find interface name
```bash
ifconfig | grep -B1 "status: active"
# Look for USB Ethernet adapter
```

### Monitor EtherCAT frames
```bash
sudo tcpdump -i en7 -XX ether proto 0x88a4
```
