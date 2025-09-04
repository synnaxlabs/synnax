#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from playwright.sync_api import Page


@pytest.fixture(autouse=True)
def setup_timeouts(page: Page):
    """Set default timeouts for all tests"""
    # Set default timeout for actions (click, type, etc.)
    page.set_default_timeout(2000)  # 2s

    page.set_default_navigation_timeout(30000)  # 30s