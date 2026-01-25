# Hardware Simulators

This directory contains reusable hardware simulators for testing control sequences
without real hardware. All simulators extend the `SimDAQ` base class which provides:

- Thread-based lifecycle management (`start()` / `stop()`)
- Automatic end command handling via a separate watcher thread
- Verbose logging mode for standalone execution
- Command-line interface for running standalone

## Available Simulators

### PressSimDAQ (`press.py`)

Simulates a pressurization system with:

- Press valve (`press_vlv_cmd`) - opens to increase pressure
- Vent valve (`vent_vlv_cmd`) - opens to decrease pressure
- Pressure transducer (`press_pt`) - reads current pressure

```bash
uv run python -m examples.simulators.press --help
```

### ThermalSimDAQ (`thermal.py`)

Simulates a thermal system with:

- Heater (`heater_cmd`) - turns on to increase temperature
- Temperature sensor (`temp_sensor`) - reads current temperature
- Force overheat command (`force_overheat_cmd`) - simulates runaway heating

```bash
uv run python -m examples.simulators.thermal --help
```

### TPCSimDAQ (`tpc.py`)

Simulates a rocket engine tank pressurization control system with:

- Oxidizer (OX) and fuel tank valves
- Pressurization system with gas booster
- Multiple pressure transducers and thermocouples

```bash
uv run python -m examples.simulators.tpc --help
```

## Creating Custom Simulators

To create a custom simulator, extend the `SimDAQ` base class:

```python
from examples.simulators.base import SimDAQ
import synnax as sy

class MySimDAQ(SimDAQ):
    description = "My custom simulator"
    end_cmd_channel = "end_my_test_cmd"  # Optional: auto-stop on this channel

    def _create_channels(self) -> None:
        # Create your Synnax channels here
        self.my_channel = self.client.channels.create(
            name="my_channel",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

    def _run_loop(self) -> None:
        loop = sy.Loop(sy.Rate.HZ * 100)
        while self._running and loop.wait():
            # Your simulation logic here
            pass


if __name__ == "__main__":
    MySimDAQ.main()
```

## Usage in Tests

Simulators can be used programmatically in tests:

```python
import synnax as sy
from examples.simulators.press import PressSimDAQ

client = sy.Synnax()
sim = PressSimDAQ(client, verbose=True)
sim.start()

# Run your test...

sim.stop()
```
