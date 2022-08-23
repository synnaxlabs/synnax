from dataclasses import dataclass

from delta import telem


@dataclass
class Channel:
    key: str = ""
    name: str = ""
    node_id: int = 0
    rate: telem.Rate = telem.Rate(0)
    data_type: telem.DataType = telem.DATA_TYPE_UNKNOWN
