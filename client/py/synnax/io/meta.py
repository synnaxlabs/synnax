from typing import Any

from pydantic import BaseModel


class ChannelMeta(BaseModel):
    """General channel metadata that can be read from a file.
    """
    name: str
    """The name of the channel."""
    meta_data: dict[str, Any]
    """Any additional metadata associated with the channel."""
