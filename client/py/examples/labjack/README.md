# LabJack Examples

This directory contains example scripts for working with LabJack devices in Synnax.

## Prerequisites

1. **Hardware**: A LabJack device (T4, T7, or T8) connected to your computer via USB,
   Ethernet, or WiFi
2. **Software**:
   - LabJack LJM driver installed (download from
     [LabJack downloads](https://labjack.com/pages/support?doc=/software-driver/installer-downloads/ljm-software-installers-t4-t7-digit/))
   - Synnax server running
   - Synnax driver running
3. **Authentication**: Logged in to Synnax CLI (`uv run sy login`)

## Quick Start Guide

Follow these scripts in order:

### 1. Connect Your Device

First, register your LabJack device with Synnax:

```bash
uv run python examples/labjack/connect_device.py
```

This script will:

- Check if the device is already registered
- Register the device with the embedded Synnax rack
- Set up the device configuration

**Configuration**: Edit the constants at the top of `connect_device.py` to match your
device:

- `DEVICE_NAME`: A friendly name for your device
- `MODEL`: Device model (`labjack.T4`, `labjack.T7`, or `labjack.T8`)
- `IDENTIFIER`: How to find your device (`"ANY"`, serial number, IP address, or device
  name)
- `CONNECTION_TYPE`: Connection method (`"ANY"`, `"USB"`, `"TCP"`, `"ETHERNET"`, or
  `"WIFI"`)

### 2. Read Data from Analog Inputs

Read analog voltages from your LabJack:

```bash
uv run python examples/labjack/read_task.py
```

This example:

- Creates channels for timestamps and two analog inputs (AIN0, AIN1)
- Configures a read task sampling at 100 Hz, streaming at 25 Hz
- Displays live voltage readings
- Press Ctrl+C to stop

**What you'll see**: Real-time voltage readings from AIN0 and AIN1.

### 3. Read Thermocouple Data

For temperature measurements with K-type thermocouples:

```bash
uv run python examples/labjack/thermocouple_read_task.py
```

This example:

- Creates channels for two K-type thermocouples (AIN0, AIN2)
- Uses device cold junction compensation (CJC)
- Samples at 10 Hz (thermocouples are slower than analog inputs)
- Displays temperature in Celsius and Fahrenheit
- Press Ctrl+C to stop

**Hardware setup**: Connect K-type thermocouples to AIN0 and AIN2 on your T7/T7-Pro.

### 4. Control Outputs

Send commands to analog and digital outputs:

```bash
uv run python examples/labjack/write_task.py
```

This example:

- Controls an analog output (DAC0) with a sine wave
- Toggles a digital output (FIO4) on/off
- Runs for 10 seconds with 1 second intervals
- Tracks command and state channels

**Hardware setup**: Connect an LED to FIO4 or an oscilloscope to DAC0 to see the
outputs.

### 5. Delete Device (Cleanup)

When finished, remove the device registration:

```bash
uv run python examples/labjack/delete_device.py
```

This will remove the device and all associated tasks from Synnax.

## Common Device Configurations

### USB Connection (Default)

```python
IDENTIFIER = "ANY"
CONNECTION_TYPE = "USB"
```

The driver will connect to any LabJack device found on USB.

### Specific Serial Number

```python
IDENTIFIER = "470012345"  # Replace with your serial number
CONNECTION_TYPE = "ANY"
```

Connect to a specific device by serial number.

### Ethernet/WiFi Connection

```python
IDENTIFIER = "192.168.1.100"  # Replace with your device IP
CONNECTION_TYPE = "ETHERNET"  # or "WIFI"
```

Connect to a network-connected LabJack.

## Channel Types

### Read Channels

- **AIChan**: Analog input (voltage)
  - Configurable range (±0.01V to ±10V)
  - Single-ended or differential
  - Use for general voltage measurements

- **ThermocoupleChan**: Thermocouple input
  - Supports types: B, E, J, K, N, R, S, T, C
  - Cold junction compensation (CJC)
  - Temperature units: Kelvin, Celsius, Fahrenheit
  - Use for temperature measurements

- **DIChan**: Digital input
  - Binary state (0 or 1)
  - Use for switches, sensors, etc.

### Write Channels

- **OutputChan** (type="AO"): Analog output
  - Voltage control (typically 0-5V on DAC0/DAC1)
  - Use for analog control signals

- **OutputChan** (type="DO"): Digital output
  - Binary control (0 or 1)
  - Use for relays, LEDs, digital control

## Sample Rates

- **Analog inputs**: Up to 100 kHz (device dependent)
- **Thermocouples**: Typically 10 Hz maximum (due to CJC calculations)
- **Digital I/O**: Up to 50 kHz

**Stream Rate**: Set lower than sample rate to buffer samples before streaming to
Synnax. For example, 100 Hz sampling with 25 Hz streaming sends 4 samples per packet.

## Troubleshooting

### "Device not found"

- Ensure the device is physically connected
- Check the LJM driver is installed
- Try `IDENTIFIER = "ANY"` to find any available device
- Check device power and USB cable

### "Failed to connect device"

- Verify the Synnax driver is running
- Check that no other software is using the device
- Try unplugging and reconnecting the device

### "Task configuration failed"

- Ensure the device was connected successfully first
- Verify channel ports match your hardware (AIN0, DAC0, FIO4, etc.)
- Check that sample rates are within device limits

### Thermocouple readings seem wrong

- Verify thermocouple type matches your hardware (K, J, T, etc.)
- Check CJC configuration (typically use `TEMPERATURE_DEVICE_K`)
- Ensure proper thermocouple connections (correct polarity)

## Additional Resources

- [LabJack T-Series Datasheet](https://labjack.com/pages/support?doc=/datasheets/t-series-datasheet/)
- [Synnax LabJack Driver Documentation](https://docs.synnaxlabs.com/reference/driver/labjack/)
- [LabJack Support Forum](https://labjack.com/forums)

## Next Steps

After running these examples, you can:

- Create custom tasks in the Synnax Console
- Build real-time dashboards with your LabJack data
- Integrate LabJack control into automated sequences
- Export data for analysis in Python, MATLAB, or other tools
