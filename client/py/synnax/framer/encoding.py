from synnax.channel.payload import (
    ChannelKey
)

class Encoding:
    # channelKeys - list[int]
    # datatypes - list[str]
    keys: list[int]
    dtypes: list[CrudeDataType]
    def __init__(
        self,
        channelKeys: list[int],
        dataTypes: list[CrudeDataType]
    ):
        self.keys = channelKeys
        self.dtypes = dataTypes

    def encoder(self, src: Frame):

    def decoder(self, src: bytearray):
