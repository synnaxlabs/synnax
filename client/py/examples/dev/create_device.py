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

racks = client.racks.retrieve()
rack_key = racks[0].key if isinstance(racks, list) else racks.key
print(f"Using rack: {rack_key}")

client.devices.create(
    [
        sy.Device(
            key="130227d9-02aa-47e4-b370-0d590add1bc1",
            rack=rack_key,
            name="PXI-6255",
            make="NI",
            model="PXI-6255",
            location="dev1",
        ),
        sy.Device(
            key="labjack-t4",
            rack=rack_key,
            name="LabJack T4",
            make="LabJack",
            model="LJM_dtT4",
            location="dev2",
        ),
        sy.Device(
            key="labjack-t7",
            rack=rack_key,
            name="LabJack T7",
            make="LabJack",
            model="LJM_dtT7",
            location="dev3",
        ),
        sy.Device(
            key="labjack-t8",
            rack=rack_key,
            name="LabJack T8",
            make="LabJack",
            model="LJM_dtT8",
            location="dev4",
        ),
        sy.Device(
            key="a0e37b26-5401-413e-8e65-c7ad9d9afd70",
            rack=rack_key,
            name="USB-6000",
            make="NI",
            model="USB-6000",
            location="dev3",
        ),
    ]
)

# Create a cDAQ chassis with nested modules to demonstrate parent_device
# relationships. The chassis appears as an expandable node in the Console's
# resource tree with the modules nested underneath it.
chassis = client.devices.create(
    sy.device.Device(
        key="sim-cdaq-9178",
        rack=rack_key,
        name="cDAQ-9178 (Sim)",
        make="NI",
        model="cDAQ-9178",
        location="Slot 1",
        properties={"is_chassis": True, "is_simulated": True, "resource_name": "cDAQ1"},
    )
)
print(f"Created chassis: {chassis.name} (key={chassis.key})")

modules = [
    ("sim-ni-9205", "NI 9205", "cDAQ-9178 Analog Input", "cDAQ1Mod1"),
    ("sim-ni-9263", "NI 9263", "cDAQ-9178 Analog Output", "cDAQ1Mod2"),
    ("sim-ni-9401", "NI 9401", "cDAQ-9178 Digital I/O", "cDAQ1Mod3"),
]

for key, model, location, resource_name in modules:
    mod = client.devices.create(
        sy.device.Device(
            key=key,
            name=f"{model} (Sim)",
            make="NI",
            model=model,
            location=location,
            rack=rack_key,
            parent_device=chassis.key,
            properties={"is_simulated": True, "resource_name": resource_name},
        )
    )
    print(f"  Created module: {mod.name} (parent={mod.parent_device})")

# Create a standalone module with no parent chassis (should appear at rack level).
standalone = client.devices.create(
    sy.device.Device(
        key="sim-ni-6289",
        name="USB-6289 (Sim)",
        make="NI",
        model="USB-6289",
        location="USB Bus",
        rack=rack_key,
        properties={"is_simulated": True, "resource_name": "Dev1"},
    )
)
print(f"  Created standalone: {standalone.name} (parent={standalone.parent_device!r})")
