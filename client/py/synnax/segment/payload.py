from freighter import Payload

from synnax.telem import TimeStamp


class SegmentPayload(Payload):
    """A payload container that represent a segment of data exchanged to and from the
    Synnax server.
    """

    data: bytes = b""
    channel_key: str = ""
    start: TimeStamp = TimeStamp(0)
