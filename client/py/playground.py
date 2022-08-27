import asyncio
import time
from datetime import datetime

import numpy as np

from delta import telem
from delta.channel import Channel
from delta.segment.writer import NumpyWriter, AsyncCore
from delta.channel.client import Client
from delta.channel.client import _RetrieveRequest, _CreateRequest, _Response
from freighter.http import GETClient, POSTClient
from freighter.endpoint import Endpoint
from freighter.encoder import MsgpackEncoderDecoder, JSONEncoderDecoder
from freighter.ws import StreamClient
from asgiref.sync import async_to_sync




def main():
    ep = Endpoint("http", "localhost", 3456, "/api/v1")
    channel = Client(
        GETClient[_RetrieveRequest, _Response](ep, JSONEncoderDecoder),
        POSTClient[_CreateRequest, _Response](ep, JSONEncoderDecoder)
    )
    async_to_sync(channel.create)(Channel(
        name="test",
        node_id=1,
        rate=25 * telem.HZ,
        data_type=telem.FLOAT64,
    ), 1)
    async_to_sync(channel.create)(Channel(
        name="test",
        node_id=2,
        rate=25 * telem.HZ,
        data_type=telem.FLOAT64,
    ), 1)
    time.sleep(1.5)
    writer = NumpyWriter(
        ["2-1", "1-1"],
        transport=StreamClient(encoder=MsgpackEncoderDecoder, endpoint=ep),
        channel_client=channel,
    )
    t0 = telem.now()
    t1 = datetime.now()
    for i in range(100):
        writer.write("2-1", np.random.rand(100000), t0 + (i * telem.TimeStamp(telem.Rate(25).span(100000))))
        writer.write("1-1", np.random.rand(100000), t0 + (i * telem.TimeStamp(telem.Rate(25).span(100000))))
    writer.close()
    print(f"{datetime.now() - t1}")

main()