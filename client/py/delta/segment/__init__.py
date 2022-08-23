from dataclasses import dataclass

import numpy

from delta import telem
from delta.telem import TimeStamp


@dataclass
class Segment:
    channel_key: str = ""
    start: telem.TimeStamp = telem.TimeStamp(0)
    data: bytes = b""
