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
        console = self.console
        client = self.client
    
        unique_id = str(uuid.uuid4())[:8]  # First 8 chars of UUID
        INDEX_NAME = f"{self.name}_{unique_id}_index_channel"

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
        console.channels.create(
            name=INDEX_NAME,
            is_index=True,
        )
        index_ch = self.client.channels.retrieve(INDEX_NAME)
        assert index_ch.data_type == sy.DataType.TIMESTAMP

        # Then, create a channel for each data type
        for data_type in data_types:
            ch_name = f"{self.name}_{unique_id}_{str(data_type)}_ch"
            console.channels.create(
                name=ch_name,
                data_type=data_type,
                is_index=False,
                index=INDEX_NAME,
            )
            time.sleep(0.2)
            ch = client.channels.retrieve(ch_name)
            assert data_type == ch.data_type
