#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from synnax.access.role.client import Client
from synnax.access.role.types_gen import ONTOLOGY_TYPE, Role, ontology_id
from synnax.util.deprecation import deprecated_getattr

_DEPRECATED = {
    "RoleClient": "Client",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = ["Role", "Client", "ONTOLOGY_TYPE", "ontology_id"]
