from synnax.transport import Transport
from . import segment
from . import channel
from freighter.endpoint import Endpoint


class Client:
    transport: Transport
    data: segment.Client
    channel: channel.Client

    def __init__(
            self,
            host: str,
            port: int,
    ):
        self.transport = Transport(Endpoint(host, port))
        self.channel = channel.Client(self.transport)
        self.data = segment.Client(self.transport, self.channel)
