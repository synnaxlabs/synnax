from pydantic import BaseModel


class SynnaxOptions(BaseModel):
    """Options class for the Synnax client."""

    host: str
    port: int
    username: str = ""
    password: str = ""
    secure: bool = False
