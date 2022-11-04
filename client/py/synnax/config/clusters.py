import keyring
from .file import ConfigFile
from pydantic import BaseModel

from synnax import SynnaxOptions


class ClusterConfig(BaseModel):
    options: SynnaxOptions


class ClustersConfig:
    internal: ConfigFile

    def __init__(self, config: ConfigFile):
        self.internal = config

    def get(self, key: str = "default") -> ClusterConfig | None:
        c = self.internal.get(f"clusters.{key}")
        if c is not None:
            c["options"]["password"] = keyring.get_password("synnax", key)
        return ClusterConfig(**c)

    def set(self, c: ClusterConfig, key: str = "default"):
        p = c.dict()
        keyring.set_password("synnax", key, p["options"].pop("password"))
        self.internal.set(f"clusters.{key}", p)
