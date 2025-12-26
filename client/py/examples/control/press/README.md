# Pressurization Control Sequence

This directory contains a control sequence that:

1. Opens a press valve (`press_vlv`) to pressurize a system in steps of 20 psi up to 100
   psi.
2. Opens a vent valve (`vent_vlv`) to vent the system.

You can also watch a [video walkthrough](https://www.youtube.com/watch?v=OJtVBfRwooA) of
this example.

## Files

This directory contains two files:

- `control_sequence.py`: The control sequence that runs the pressurization and venting
  process.
- `simulated_daq.py`: A data acquisition computer that simulates how a real
  pressurization system would respond to the commands issued by the control sequence.

## Running the Example

To run the example, make sure you have:

1. A Synnax Core running locally. See the
   [Quick Start](https://docs.synnaxlabs.com/reference/core/quick-start) guide for more
   information.
2. The Synnax Python client installed. See the
   [Quick Start](https://docs.synnaxlabs.com/reference/client/quick-start)
   guide for more information.
3. Optionally, the Synnax Console installed and set up to visualize the data generated
   by the control sequence. See the
   [Getting Started](https://docs.synnaxlabs.com/reference/console/get-started) guide
   for more information.

If you are using the Console, you can use a combination of
[Schematics](https://docs.synnaxlabs.com/reference/console/schematics) and
[Line Plots](https://docs.synnaxlabs.com/reference/console/line-plots) to visualize the
state of the system over time.

Once you have a Synnax cluster running, start the simulated DAQ by running the following
command:

```bash
python simulated_daq.py
```

Then, run the control sequence by running the following command:

```bash
python control_sequence.py
```

The control sequence will run and you should see the pressure in the system increase and
decrease over time. You should also see the state of the pressurization system update in
the Console.
