"""
Run first prior to running bench_latency.py
"""

import gc

import synnax as sy

gc.disable()

# client = sy.Synnax()
client = sy.Synnax(
            host='localhost',
            port=9090,
            username='synnax',
            password='seldon',
            secure=False,
        )

STATE_CHANNEL = "state"
CMD_CHANNEL = "command"

STATE = True

times = list()

loop_start = sy.TimeStamp.now()

client.channels.create(
    name=STATE_CHANNEL,
    data_type=sy.DataType.UINT16,
    virtual=True,
    retrieve_if_name_exists=True,
)

client.channels.create(
    name=CMD_CHANNEL,
    data_type=sy.DataType.UINT16,
    virtual=True,
    retrieve_if_name_exists=True,
)

with client.open_streamer(CMD_CHANNEL) as stream:
    with client.open_writer(sy.TimeStamp.now(), STATE_CHANNEL) as writer:
        while True:
            frame = stream.read()
            writer.write(STATE_CHANNEL, frame[CMD_CHANNEL])