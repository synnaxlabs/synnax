#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.ranger.client import Client, Range
from synnax.ranger.retrieve import Retriever
from synnax.ranger.writer import Writer
from synnax.util.deprecation import deprecated_getattr

_DEPRECATED = {
    "RangeRetriever": "Retriever",
    "RangeWriter": "Writer",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = ["Client", "Range", "Retriever", "Writer"]
