import multiprocessing
import random
import time
import sys
from typing import NamedTuple, List

import numpy as np
import synnax as sy


class SetUpConfig(NamedTuple):
    num_index: int
    num_data: int


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


def create_channels(tc: SetUpConfig):
    channels = []
    for i in range(tc.num_index):
        index = client.channels.create(
            name="int" + str(i),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        channels.append(index.key)

    num_data_channels_per_index = tc.num_data // tc.num_index
    for ind in range(tc.num_index):
        for k in range(
            num_data_channels_per_index,
        ):
            ch = client.channels.create(
                name=f"int{ind}-{k}",
                index=channels[ind],
                data_type=sy.DataType.FLOAT32,
                retrieve_if_name_exists=True,
            )
            channels.append(ch.key)


if __name__ == "__main__":
    if len(sys.argv) != 3:
        exit(-1)
    tc = SetUpConfig(int(sys.argv[1]), int(sys.argv[2]))
    create_channels(tc)
