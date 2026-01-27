# Abort Sequence Example

This example demonstrates how to implement an abort sequence that monitors for
overpressure conditions and takes control to safely vent the system.

## Files

This directory contains:

- `abort_sequence.py`: Monitors pressure and takes control when it exceeds 30 psi.
- `nominal_sequence.py`: A normal pressurization sequence that runs alongside the abort.

The simulator for this example is located in `examples/simulators/press.py`.

## Running the Example

To run this example, you'll need three terminals open.

**Terminal 1** - Start the simulator:

```bash
cd client/py
uv run python -m examples.simulators.press
```

**Terminal 2** - Start the abort sequence listener (must start before nominal):

```bash
cd client/py
uv run python -m examples.control.abort.abort_sequence
```

**Terminal 3** - Run the nominal sequence:

```bash
cd client/py
uv run python -m examples.control.abort.nominal_sequence
```

We recommend using the
[Synnax Console](https://docs.synnaxlabs.com/reference/console/get-started) to visualize
the data in these examples.
