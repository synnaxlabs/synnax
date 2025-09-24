# Abort Sequence Example

To run this example, you'll need three Python shells open. In the first shell, start the
simulated data acquisition device:

```bash
python simulated_daq.py
```

In the second shell, start the abort sequence listener:

```bash
python abort_sequence.py
```

In the third shell, run the nominal sequence:

```bash
python nominal_sequence.py
```

We recommend using the
[Synnax Console](https://docs.synnaxlabs.com/reference/console/get-started) to visualize
the data in these examples.
