#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
import uuid
from test.console.console_case import ConsoleCase

import synnax as sy


class Create_Channel_Types(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:
        """
        Test the "create a channel" modal for all data types
        """
        self._log_message("Running Create_Channel_Types")
        console = self.console
        client = self.client

        INDEX_NAME = "index_channel"

        data_types = [
            sy.DataType.FLOAT64,
            sy.DataType.FLOAT32,
            sy.DataType.INT64,
            sy.DataType.INT32,
            sy.DataType.INT16,
            sy.DataType.INT8,
            sy.DataType.UINT64,
            sy.DataType.UINT32,
            sy.DataType.UINT16,
            sy.DataType.UINT8,
            sy.DataType.UUID,
        ]

        # First, create an index channel
        if console.channels.create(
            name=INDEX_NAME,
            is_index=True,
        ):
            self._log_message(f"Created index channel: {INDEX_NAME}")
        else:
            self._log_message(f"Index channel already exists: {INDEX_NAME}")

        index_ch = self.client.channels.retrieve(INDEX_NAME)
        assert index_ch.data_type == sy.DataType.TIMESTAMP
        self._log_message(f"Created index channel: {INDEX_NAME}")
        # Then, create a channel for each data type
        for data_type in data_types:
            ch_name = f"{str(data_type)}_ch"
            if console.channels.create(
                name=ch_name,
                data_type=data_type,
                is_index=False,
                index=INDEX_NAME,
            ):
                self._log_message(f"Created channel: {ch_name}")
            else:
                self._log_message(f"Channel already exists: {ch_name}")

            time.sleep(1)
            ch = client.channels.retrieve(ch_name)
            assert data_type == ch.data_type
