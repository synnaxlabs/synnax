# Hardware Simulators

This directory contains reusable hardware simulators for testing control sequences
without real hardware. There are two types of simulators:

1. **SimDAQ** - Thread-based simulators that write directly to Synnax channels
2. **DeviceSim** - Process-based simulators that expose network protocol endpoints

Both extend the shared `Simulator` base class which provides `start()` / `stop()`
lifecycle management and verbose logging.

## SimDAQ Simulators

SimDAQ simulators create Synnax channels and write data directly. They provide:

- Automatic end command handling via a separate watcher thread
- Command-line interface for running standalone

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

## DeviceSim Simulators

DeviceSim simulators expose network protocol endpoints (Modbus TCP, OPC UA) for testing
driver integration. They run async servers in a subprocess and do NOT interact with
Synnax directly - the C++ driver connects to the endpoint and writes data to Synnax.

### ModbusSim (`examples/modbus/server.py`)

Runs a Modbus TCP server on port 5020 with:

- Holding registers (addresses 0-4): Sine wave data
- Input registers (addresses 0-4): Sine wave data
- Discrete inputs (addresses 0-3): Rotating binary patterns
- Coils (addresses 0-4): Static digital outputs

### OPCUASim (`examples/opcua/server.py`)

Runs an OPC UA server on port 4841 with:

- Float variables (`my_float_0` - `my_float_4`): Sine wave data
- Boolean variables (`my_bool_0` - `my_bool_4`): Square wave patterns
- Array variables (`my_array_0` - `my_array_4`): 5-element float arrays
- Command variables (`command_0` - `command_2`): Writable floats

### OPCUATLSSim (`examples/opcua/server.py`)

Runs a TLS-encrypted OPC UA server on port 4842 with `Basic256Sha256_SignAndEncrypt`
security. Exposes the same variables as `OPCUASim`. Self-signed server and client
certificates are generated automatically under `examples/opcua/certificates/`.

### OPCUATLSAuthSim (`examples/opcua/server.py`)

Runs a TLS-encrypted OPC UA server on port 4843 with `Basic256Sha256_SignAndEncrypt`
security and username/password authentication (`testuser` / `testpass`). Exposes the
same variables as `OPCUASim`.

## Creating Custom Simulators

### Custom SimDAQ

Extend the `SimDAQ` base class for simulators that write directly to Synnax:

```python
from examples.simulators import SimDAQ
import synnax as sy

class MySimDAQ(SimDAQ):
    description = "My custom simulator"
    end_cmd_channel = "end_my_test_cmd"  # Optional: auto-stop on this channel

    def _create_channels(self) -> None:
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

### Custom DeviceSim

Extend `DeviceSim` for simulators that expose a network protocol endpoint:

```python
from examples.simulators.device_sim import DeviceSim
from synnax import modbus

class MyDeviceSim(DeviceSim):
    description = "My custom device simulator"
    host = "127.0.0.1"
    port = 5020
    device_name = "My Test Device"

    async def _run_server(self) -> None:
        # Start your async server here
        ...

    @staticmethod
    def create_device(rack_key: int) -> modbus.Device:
        return modbus.Device(
            host=MyDeviceSim.host,
            port=MyDeviceSim.port,
            name=MyDeviceSim.device_name,
            rack=rack_key,
        )
```

## Usage in Tests

SimDAQ simulators can be used programmatically:

```python
import synnax as sy
from examples.simulators.press import PressSimDAQ

client = sy.Synnax()
sim = PressSimDAQ(client, verbose=True)
sim.start()

# Run your test...

sim.stop()
```

DeviceSim simulators are used via the integration test framework:

```python
from examples.modbus import ModbusSim

sim = ModbusSim()
sim.start()   # Starts server subprocess on port 5020

# Driver connects to the endpoint...

sim.stop()    # Terminates server subprocess
```
