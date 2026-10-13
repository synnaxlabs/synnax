# Turbine demo

This branch adds the Kafka integration and spectrum mode for line plots. The demo
streams turbine telemetry from Kafka into a Core and shows test-bench waveforms on a
line plot.

## Requirements

- Go, pnpm, and uv, as for any build of this repo.
- Docker, for the Kafka broker.

## 1. Start the Core and the Console

```sh
cd core && go build -o synnax . && ./synnax start -mi
```

`-mi` runs the Core in memory and without TLS. A restart clears all data, so run step 3
again after each restart.

In a second terminal:

```sh
pnpm install
pnpm turbo build --filter "@synnaxlabs/console^..."
pnpm dev:console-vite
```

Open `http://localhost:5173` and log in to the local Core as `synnax` with the password
`seldon`.

## 2. Start Kafka and the telemetry producer

```sh
docker run -d --name demo-kafka -p 9092:9092 apache/kafka:4.1.0
docker exec demo-kafka /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server localhost:9092 --create --topic turbines
python3 demo/produce_turbines.py | docker exec -i demo-kafka \
  /opt/kafka/bin/kafka-console-producer.sh --bootstrap-server localhost:9092 \
  --topic turbines --property parse.key=true --property "key.separator=	"
```

The separator in the last command is a tab character. Leave the producer running.

## 3. Connect Kafka to the Core

```sh
uv run --project client/py python demo/setup_kafka.py
```

This creates the "Plant" Kafka cluster and starts the "Turbines" read task. Each turbine
writes `plant_turbine_<a|b>_rpm`, `_exhaust_temp`, and `_vibration`. Add `--port` if the
Core does not run on 9090.

## 4. Start the waveforms

```sh
uv run --project client/py python client/py/examples/dev/waveforms.py
```

This streams `waveform_sine`, `waveform_noisy`, `waveform_square`, and `waveform_sweep`
at 2 kHz.

## What to show

- **Kafka:** open the "Turbines" task to show its records and fields. Then plot
  `plant_turbine_a_rpm` and `plant_turbine_b_rpm`.
- **Measurements:** open the Measure tab to show live statistics for each line.
- **Spectrum:** on a line plot with a rolling range, set the mode to Spectrum. The sine
  shows peaks at 60 Hz and 180 Hz. The square wave shows a 20 Hz fundamental and its
  harmonics.
