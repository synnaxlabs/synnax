# Dewesoft modules

Facts about Dewesoft IOLITE, KRYPTON, and SIRIUS modules that the code does not show.
Each one was found on hardware or in the Dewesoft manual for third-party EtherCAT
masters. The user page is
`site/docs/src/pages/reference/hardware/ethercat/dewesoft.mdx`.

The Driver reads Dewesoft settings from a catalog that the `synnaxlabs/esi` parser makes
from its supplements. The catalog compiles into
`driver/ethercat/esi/registry_settings.inc`. Never edit that file. Change the
supplement, then generate it again.

## Modes

Each module stores the master that it sends data to. The catalog calls this object
`mode`.

- **`0x8020:09`** on the IOLITE modules that have a dual mode, `0x8020:0A` on
  product 254. `0` is Synnax only with DC sync on bus 1. `3`, where listed, is Synnax
  only with SM sync. `1` is dual: DewesoftX on bus 1, and the Driver reads the same data
  on bus 2 (`shared_with`).
- **`0x8020:07`** on products 100 to 162. A value with `only_read_by` sends data that
  only DewesoftX reads. Synnax gets nothing, and the scan says so.
- **No mode** in the catalog for the 32xDO (product 251) and product 247.

On `0x8020:09` and `0x8020:0A`, the catalog marks bus 2 with a revision bit,
`second_bus` (`0x1000`), so the scan finds the bus without a read. Entries on
`0x8020:07` have no such bit.

A new mode takes effect only at the next power restart. Until then the module keeps the
old mode, and the scan shows "Waiting for a power restart".

## Store to flash

On modules that have it, `0x8020:02` is PowerONdefault. A write of `2` stores the mode
in flash, and the object then reads `1`. The catalog marks this as `save`, and **Switch
to Synnax** writes the mode, then the save.

- Proven 2026-11-17: a stored dual mode and a Driver switch to Synnax only both survive
  a full power loss.
- The 32xDO (revision 5) has no PowerONdefault: its `0x8020:02` is "Output Inverted". On
  the 8xDI-4xDO it is "Inverted outputs". Never add a `save` to a model because a
  similar model has one.
- The 32xDO keeps a mode written without a save through a full unplug.
- Write PowerONdefault only through the switch, because each write goes to flash.

## Identity

`0x1018:04` is an 8-byte serial (`UNSIGNED64`) on every IOLITE module. A 4-byte read
fails with "the reply is larger than the read allows". The device key ends in the
serial. The SOEM master drops a serial above 2^53 - 1, the largest whole number that
JSON keeps exactly.

Proven 2026-11-17: when two modules swap slots or a module moves, each device keeps its
name, settings, and tasks. A task binds to its slave by serial at start.

The serial is the same on both buses, so a cable move from bus 1 to bus 2 keeps every
device. The slave name and the PDO layout change with the bus:

| Module     | Bus 1 name                     | Bus 2 name             | On bus 2                     |
| ---------- | ------------------------------ | ---------------------- | ---------------------------- |
| GATE LATCH | `IOLITE GATE LATCH v2.00 BUS1` | `IOLiteGATE_LATCH2.00` | 26 inputs, 6 outputs (27/13) |
| 6xSTG      | `IOLiteSTG_KK`                 | `IOLITEr 6xSTG`        | Settings read fails          |
| 32xDO      | `IOLITEr 32xDO BUS1`           | `IOLITEr 32xDO BUS2`   | 32 outputs                   |

On bus 2, a Synnax only 6xSTG has no `0x8050:03`, so its settings read fails. A start
then fails with "Fatal sync error" in SAFE-OP and gives no advice (SY-6230).

## Sync

- **DC by default.** Synnax only IOLITE modules run DC sync. Some also list SM sync, as
  value `3` with `assign_activate` `0`.
- **6xSTG timing.** `assign_activate` `0x0300`, SYNC0 1 µs, SYNC1 1 s. Most other
  modules use `0x0700`. The SYNC1 register holds T1 - T0 plus the shift, so 1 s with a 1
  ms SYNC0 and no shift reads 0.999 s.
- **GATE buses.** The GATE lists one DC mode for each bus. The mode for the other bus
  refuses SAFE-OP with "Invalid DC SYNC configuration".
- **16xAO and 16xLV.** Their ESI SYNC1 is 1000 bus cycles. It fits only at 233 Hz or
  more, and it equals the manual's 1 s only at 1 kHz. Users set the cycle by hand.
- **Held process data.** If the master reads the 6xSTG inputs while the module moves
  from PRE-OP to SAFE-OP, its firmware never starts the data loop, and the inputs stay
  `0`. The SOEM master holds process data until SAFE-OP and sends only the reference
  clock frame (`process_data_held` in `soem/master.cpp`). Keep the hold.
- **Unused sync managers.** A Synnax only 6xSTG reaches SAFE-OP only when its unmapped
  SM2, SM4, and SM5 are off. Else it refuses with "Invalid output configuration".

## Rack slices

- IOLITEr slices are not hot swappable. Turn the rack off first.
- Each slice has two M2.5x5 Torx panhead screws, one at the top and one at the bottom of
  its front panel. The slice does not slide on rails.
- An empty slot breaks the chain, and the rack hides every slice to its right. Close the
  gap, or fill the slot with an IOLITEr-BLANK.
- Seen 2026-11-17 when the last slice was pulled while tasks ran: the write task warned
  "Lost contact with slave 7" at once. The read task kept running and lost 75 samples in
  one 61 ms gap. When the slice went back, it reached OP in about 3 s with no lost
  samples.

## 32xDO revisions

Two 32xDO revisions on the station differ in layout. Revision 5 has 32 outputs. The
other adds 7: Flag1 to Flag4, RTRCount, RTR8count, and WritePacketNumber. Both have the
LED outputs that the nightly write test drives.

## 6xSTG amplifier settings

- `0x8040+n` holds the settings of channel `n`, `0x8050+n` its range table, `0x8060+n`
  its filter table, and `0x8070+n` its excitation limits.
- Writes apply at once, also in OP. A range write makes the module compute its own
  gains, so the Driver does no gain math.
- The module does not check values. A range of `5` silently became `0`.
- A bridge mode at 0 V excitation reads inf or NaN.
- The same objects hold factory gains and offsets, and these are writable. Never show or
  write them.

## Dual mode limits

- DewesoftX drives the digital outputs of a dual module. SY-5859 tracks Synnax control.
- One converter feeds both masters, and DewesoftX sets its rate. Bus 2 data gets no
  added anti-aliasing filter.

## DewesoftX on the same computer

DewesoftX adds the NDIS filter `DW_EthcatDAQ` to each Ethernet adapter. The filter
blocks the Driver, also when DewesoftX is closed. Turn it off on the adapter of the
Driver only:

```powershell
Disable-NetAdapterBinding -Name "Ethernet 2" -ComponentID DW_EthcatDAQ
```

## Open defects

- SY-6201: after the whole bus loses power, the DC restore of the GATE LATCH stays about
  781 ms off the reference clock.
- SY-6038: a 32xDO can leave OP on a sync manager watchdog at start.
- SY-6230: no bus 1 advice for Synnax only modules on bus 2.
