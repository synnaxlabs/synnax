from freighter import URL
from pydantic import BaseModel

from .auth import AuthenticationClient

from .channel import ChannelClient
from .channel.create import ChannelCreator
from .channel.registry import ChannelRegistry
from .channel.retrieve import ChannelRetriever
from .segment import SegmentClient

from .transport import Transport


class SynnaxOptions(BaseModel):
    """Options class for the Synnax client.
    """
    host: str
    port: int
    username: str = ""
    password: str = ""


class Synnax:
    """Client to perform operations against a Synnax cluster. If no credentials are provided
    in the options, the client will attempt to load them from the configuration file (
    ~/.synnax/config.json) or from environment variables.

    :param host: Hostname of a Synnax server.
    :param port: Port of a Synnax server.
    :param username: Username to authenticate with. Not required if the Synnax cluster
    is running in insecure mode.
    :param password: Password to authenticate with. Not required if the Synnax cluster
    is running in insecure mode.
    """

    _transport: Transport
    channel: ChannelClient
    data: SegmentClient

    def __init__(
        self,
        host: str = "",
        port: int = 0,
        username: str = "",
        password: str = "",
    ):
        self._transport = Transport(URL(host=host, port=port))
        if username != "" or password != "":
            auth = AuthenticationClient(
                self._transport.http.post_client(), username, password
            )
            auth.authenticate()
            self._transport.use(*auth.middleware())
        ch_retriever = ChannelRetriever(self._transport.http)
        ch_creator = ChannelCreator(self._transport.http)
        ch_registry = ChannelRegistry(ch_retriever)
        self.data = SegmentClient(self._transport, ch_registry)
        self.channel = ChannelClient(self.data, ch_retriever, ch_creator)
