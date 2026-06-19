#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any


def none_to_empty(v: Any) -> Any:
    """Coerce None to an empty dict for required record fields.

    The wire treats a null record the same as an absent one: no entries. Used as a
    pydantic BeforeValidator so an explicit null validates as {} instead of raising.
    """
    return {} if v is None else v
