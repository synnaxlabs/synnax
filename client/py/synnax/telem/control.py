#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# Backwards compatibility re-exports. Canonical definitions live in synnax.x.control.
from synnax.x.control import Authority, CrudeAuthority, Subject

__all__ = ["Authority", "CrudeAuthority", "Subject"]
