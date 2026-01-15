#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

rack = client.racks.create(name="NI / LabJack Test Rack")

# Create a cDAQ chassis device first - this will be the parent of the modules
chassis = sy.Device(
    key="cdaq-9178-chassis",
    rack=rack.key,
    name="cDAQ-9178 Chassis",
    make="NI",
    model="cDAQ-9178",
    location="cDAQ1",
)
client.devices.create(chassis)
print(f"Created chassis: {chassis.name} (key: {chassis.key})")

# Create NI modules as children of the chassis using parent_device
# This establishes an ontology relationship: chassis -> modules
modules = [
    sy.Device(
        key="ni-9205-mod1",
        rack=rack.key,
        name="NI 9205 - Analog Input",
        make="NI",
        model="NI 9205",
        location="cDAQ1Mod1",
        parent_device=chassis.key,  # Module is a child of the chassis
    ),
    sy.Device(
        key="ni-9263-mod2",
        rack=rack.key,
        name="NI 9263 - Analog Output",
        make="NI",
        model="NI 9263",
        location="cDAQ1Mod2",
        parent_device=chassis.key,
    ),
    sy.Device(
        key="ni-9401-mod3",
        rack=rack.key,
        name="NI 9401 - Digital I/O",
        make="NI",
        model="NI 9401",
        location="cDAQ1Mod3",
        parent_device=chassis.key,
    ),
]
client.devices.create(modules)
print(f"Created {len(modules)} modules as children of chassis")

# Also create some standalone devices (no parent_device - direct children of rack)
standalone_devices = [
    sy.Device(
        key="130227d9-02aa-47e4-b370-0d590add1bc1",
        rack=rack.key,
        name="PXI-6255",
        make="NI",
        model="PXI-6255",
        location="dev1",
    ),
    sy.Device(
        key="labjack-t4",
        rack=rack.key,
        name="LabJack T4",
        make="LabJack",
        model="LJM_dtT4",
        location="dev2",
    ),
    sy.Device(
        key="labjack-t7",
        rack=rack.key,
        name="LabJack T7",
        make="LabJack",
        model="LJM_dtT7",
        location="dev3",
    ),
]
client.devices.create(standalone_devices)
print(f"Created {len(standalone_devices)} standalone devices")

print("\nDevice hierarchy:")
print(f"  Rack: {rack.name}")
print(f"    └── {chassis.name} (chassis)")
for mod in modules:
    print(f"        └── {mod.name}")
for dev in standalone_devices:
    print(f"    └── {dev.name} (standalone)")
