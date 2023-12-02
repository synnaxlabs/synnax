import math

import synnax as sy
import time

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)

ANALOG_CHANNELS = range(0, 80)

ai_idx = sy.Channel(
    name="gse_ai_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True
)
doa_idx = sy.Channel(
    name="gse_doa_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True
)
ai_idx = client.channels.create(ai_idx, retrieve_if_name_exists=True)
doa_idx = client.channels.create(doa_idx, retrieve_if_name_exists=True)

ai_channels = [
    sy.Channel(
        name=f"gse_ai_{i}",
        index=ai_idx.key,
        data_type=sy.DataType.FLOAT32,
    ) for i in ANALOG_CHANNELS
]
ai_channels = client.channels.create(ai_channels, retrieve_if_name_exists=True)

DIGITAL_CHANNELS = range(0, 24)

doa_channels = [
    sy.Channel(
        name=f"gse_doa_{i}",
        index=doa_idx.key,
        data_type=sy.DataType.FLOAT32,
    ) for i in DIGITAL_CHANNELS
]
doa_channels = client.channels.create(doa_channels, retrieve_if_name_exists=True)

doc_index_channels = [
    sy.Channel(
        name=f"gse_doc_{i}_time",
        data_type=sy.DataType.TIMESTAMP,
        is_index=True
    ) for i in DIGITAL_CHANNELS
]

doc_index_channels = client.channels.create(doc_index_channels, retrieve_if_name_exists=True)

doc_channels = [
    sy.Channel(
        name=f"gse_doc_{i}",
        index=doc_index_channels[i].key,
        data_type=sy.DataType.FLOAT32,
    ) for i in DIGITAL_CHANNELS
]

doc_channels = client.channels.create(doc_channels, retrieve_if_name_exists=True)

rate = (sy.Rate.HZ * 30).period.seconds

state = {
    **{ch.key: 0 for ch in doa_channels},
    **{ch.key: 0 for ch in ai_channels},
}

doc_channels_to_ack = {cmd.key: doa_channels[i].key for i, cmd in enumerate(doc_channels)}

i = 0
with client.new_streamer([c.key for c in doc_channels]) as streamer:
    with client.new_writer(
            sy.TimeStamp.now(),
            channels=[ai_idx.key, doa_idx.key, *[c.key for c in ai_channels], *[c.key for c in doa_channels]],
            name="Writer",
    ) as writer:
        press = 0
        while True:
            time.sleep(rate)
            if streamer.received:
                f = streamer.read()
                for k in f.columns:
                    state[doc_channels_to_ack[k]] = f[k][0]

            now = sy.TimeStamp.now()
            state[ai_idx.key] = now
            state[doa_idx.key] = now

            for j, ch in enumerate(doa_channels):
                if state[ch.key] == 1:
                    if (j % 2) == 0:
                        state[ai_channels[math.floor(j /2)].key] += 1
                    else:
                        state[ai_channels[math.floor(j / 2)].key] -= 1

            ok = writer.write(state)
            if not ok:
                break
            i += 1
            if (i % 100) == 0:
                print(f"Committing {i} samples")
                print(writer.commit())
