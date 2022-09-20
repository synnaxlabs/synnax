from dataclasses import dataclass
from synnax.telem import (
    Rate,
    DataType,
    Density,
    UnparsedRate,
    UnparsedDataType,
    UnparsedDensity,
    DATA_TYPE_UNKNOWN,
)


@dataclass
class Record:
    """A pure container that represents the core properties of a channel.
    ChannelRecord provides no functionality outside property storage.
    """
    key: str
    name: str
    node_id: int
    rate: Rate
    data_type: DataType
    density: Density

    def __init__(
            self,
            name: str = "",
            node_id: int = 0,
            rate: UnparsedRate = Rate(0),
            data_type: UnparsedDataType = DATA_TYPE_UNKNOWN,
            density: UnparsedDensity = Density(0),
            key: str = "",
    ):
        self.name = name
        self.node_id = node_id
        self.rate = Rate(rate)
        self.data_type = DataType(data_type)
        self.density = Density(density)
        self.key = key
