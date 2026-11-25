# Records stored by a `core/v0.61.4` Core

Each `.hex` file is the value that a `core/v0.61.4` Core stored for one record, as hex
in lines of 88 characters. The schematic is v11, and symbols are symbol v3, with a
float alpha in each color.

To make them, the `core/v0.61.4` Python client created the records on a `core/v0.61.4`
Core with its data on disk. After the Core stopped, a read-only Pebble iterator read
each value under `gorp.Schematic` and `gorp.Symbol`.

| File                    | Holds                                                         |
| ----------------------- | ------------------------------------------------------------- |
| `custom_nodes.hex`      | Schematic "Custom nodes", with the custom nodes below         |
| `symbol_vent_valve.hex` | Actuator symbol "Vent valve": base blue at alpha 0.5, active green |
| `symbol_tank.hex`       | Static symbol "Tank": base blue at alpha 0.5                  |
| `symbol_plain.hex`      | Static symbol "Plain", with no states                         |

The nodes of "Custom nodes", each a custom_actuator node on `valve_state` and
`valve_cmd` unless noted:

| Label              | Symbol       | State overrides                                |
| ------------------ | ------------ | ---------------------------------------------- |
| `Red vent`         | Vent valve   | Active red                                     |
| `White vent`       | Vent valve   | Active with a red stroke and a white fill      |
| `Plain actuator`   | Plain        | None                                           |
| `Missing actuator` | A random key | Active red                                     |
| `Red tank`         | Tank         | Base red; a custom_static node                 |
