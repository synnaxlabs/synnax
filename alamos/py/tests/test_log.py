#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos.log import NOOP_LOGGER


class TestLog:
    def test_noop(self) -> None:
        """
        Should not raise an exception.
        """
        logger = NOOP_LOGGER
        logger.debug("test")
        logger.info("test")
        logger.warn("test")
        logger.error("test")
