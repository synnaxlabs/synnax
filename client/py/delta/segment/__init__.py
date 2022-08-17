from dataclasses import dataclass
from delta.telem import TimeStamp

@dataclass
class Segment:
    channel_key: str
    start: int
    data: bytes

