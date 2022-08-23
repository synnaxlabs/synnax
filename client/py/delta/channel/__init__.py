from dataclasses import dataclass

from delta import telem


@dataclass
class Channel:
    key: str = ""
    name: str = ""
    node_id: int = 0
    data_rate: telem.Rate = telem.Rate(0)
    data_type: telem.Density = telem.Density(0)
