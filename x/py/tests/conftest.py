#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from hypothesis import settings

# Derandomized so a CI failure reproduces on every run.
settings.register_profile("x", max_examples=200, derandomize=True, deadline=None)
settings.load_profile("x")
