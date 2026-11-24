# Records stored by a `core/v0.62.0` Core

Each `.hex` file is the value that a `core/v0.62.0` Core stored for one record, as hex
in lines of 88 characters. Schematics are v13, and symbols are symbol v4. Each `.json`
file is a schematic that the same Core exported, formatted by Prettier.

To make them, the `core/v0.62.0` Python client created the records on a `core/v0.62.0`
Core with its data on disk. After the Core stopped, a read-only Pebble iterator read
each value under `gorp.Schematic` and `gorp.Symbol`.

| File                                      | Holds                                                     |
| ----------------------------------------- | --------------------------------------------------------- |
| `equipment.hex`, `equipment.json`         | One node of each equipment variant, and their edge cases  |
| `symbol_vent_valve.hex`                   | The actuator symbol that the `custom_actuator` nodes draw |
| `symbol_tank.hex`                         | The static symbol that the `custom_static` nodes draw     |
| `valve_panel_a.hex`, `valve_panel_a.json` | Selects and state indicators with inline options          |
| `valve_panel_b.hex`, `valve_panel_b.json` | The same options in a second schematic                    |
| `select_empty_option_key.hex`             | A select option with an empty key                         |
| `select_duplicate_option_key.hex`         | Two select options with the same key                      |

A `core/v0.62.0` Core stores an option value of NaN or infinity as 0, so no file holds
one.

Do not regenerate these files. They are proof of what a stable Core stored.
