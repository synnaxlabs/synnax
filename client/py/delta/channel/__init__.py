
class ChannelKey:
    value: bytes

    def __init__(self, node_id: int, channel_id: int):
        self.value = bytes([node_id, channel_id])