from freighter import Payload

from synnax.telem import DataType, Density, Rate


class ChannelPayload(Payload):
    """A payload container that represent the properties of a channel exchanged to and
    from the Synnax server.
    """

    data_type: DataType
    density: Density
    rate: Rate
    name: str = ""
    node_id: int = 0
    key: str = ""
