# Tank Pressurization Control Sequence

This directory contains a control sequence to simulate a tank-pressure control (TPC)
test sequence. TPC sequences are commonly used to control the pressure in rocket
propellant tanks.

## Files

This directory contains several files:

- `common.py`: The names of the valves and sensors on the TPC system.
- `control_sequence.py`: The main control sequence that runs the pressurization and
  venting process. This sequence also reruns the test with more optimal control bounds.
- `simulated_daq.py`: A data acquisition computer (DAQ) that simulates how a real
  pressurization system would respond to the commands issued by the control sequence.
- `threshold_detection.py`: A script that will automatically create
  [ranges](https://docs.synnaxlabs.com/reference/concepts/ranges) when the fuel tank
  pressure rises above a certain threshold.
- `Operator.json`: An example schematic that can be imported into Console to visualize
  and control the TPC system.

## Running the Example

To run the example, make sure you have:

1. A Synnax Core running locally. See the
   [Quick Start](https://docs.synnaxlabs.com/reference/cluster/quick-start) guide for
   more information.
2. The Synnax Python client installed. See the
   [Getting Started](https://docs.synnaxlabs.com/reference/python-client/get-started)
   guide for more information.
3. Optionally, the Synnax Console installed and set up to visualize the data generated
   by the control sequence. See the
   [Getting Started](https://docs.synnaxlabs.com/reference/console/get-started) guide
   for more information.

If you are using the Console, you can use a combination of
[Schematics](https://docs.synnaxlabs.com/reference/console/schematics) and
[Line Plots](https://docs.synnaxlabs.com/reference/console/line-plots) to visualize the
state of the system over time.

Once you have a Synnax Core running, start the simulated DAQ by running the following
command:

```bash
python simulated_daq.py
```

Then, run the control sequence by running the following command:

```bash
python control_sequence.py
```

You can also optionally run the threshold detection sequence by running the following
file:

```bash
python threshold_detection.py
```

The control sequence will run and you should see the pressure in the system increase and
decrease over time. You should also see the state of the pressurization system update in
the Console.

## Using the Operator Schematic

This directory includes an example schematic (`Operator.json`) that provides a visual
interface for monitoring and controlling the example TPC system. To use it:

1. Import the schematic into Console by dragging and dropping `Operator.json` into the
   Console window, or by right-clicking on a workspace in the Workspaces menu and
   selecting "Import Schematic".
2. Once imported, you'll need to connect the schematic elements to your channels. All
   channel keys in the schematic have been left empty so you can configure them for your
   specific setup.
3. To add channel keys, right-click on each schematic element (valves, sensors,
   displays) and select "Properties". Then, select the appropriate channel from your
   Core that corresponds to each element.
4. Make sure the channel names match those defined in `common.py` (e.g., `ox_press_cmd`,
   `fuel_pt_1`, etc.).

Once configured, the schematic will display live sensor values and allow you to manually
control valves and other components in the TPC system.
