#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from console.case import ConsoleCase


class Channel_Lifecycle(ConsoleCase):
    """
    Test the lifecycle of channels
    """

    def run(self) -> None:
        """
        Test the "create a channel" modal for all data types
        """

        ch_list, data_types = self.create_channels()
        ch_renamed = self.rename_channels(ch_list, data_types)
        self.delete_channels(ch_renamed)

    def create_channels(self) -> tuple[list[sy.Channel], list[sy.DataType]]:
        self._log_message("Creating channels")
        console = self.console
        client = self.client
        ch_list = list[sy.Channel]()

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
        ch_list.append(INDEX_NAME)

        index_ch = self.client.channels.retrieve(INDEX_NAME)
        assert index_ch.data_type == sy.DataType.TIMESTAMP

        # Then, create a channel for each data type
        for data_type in data_types:
            ch_name = f"{str(data_type)}_ch"
            if console.channels.create(
                name=ch_name,
                data_type=data_type,
                index=INDEX_NAME,
            ):
                self._log_message(f"Created channel: {ch_name}")
            else:
                self._log_message(f"Channel already exists: {ch_name}")
            ch_list.append(ch_name)
            ch = client.channels.retrieve(ch_name)
            assert data_type == ch.data_type

        # Reverse so we delete the index channel last
        ch_list.reverse()
        data_types.reverse()
        data_types.append(sy.DataType.TIMESTAMP)
        return ch_list, data_types

    def rename_channels(
        self, ch_list: list[sy.Channel], data_types: list[sy.DataType]
    ) -> list[sy.Channel]:
        self._log_message("Renaming channels")

        # Rename the channels
        ch_list_renamed = [f"{ch}_renamed" for ch in ch_list]
        self.console.channels.rename(ch_list, ch_list_renamed)

        # Verify the data types
        for ch, dtype_expected in zip(ch_list_renamed, data_types):
            dtype_actual = self.client.channels.retrieve(ch).data_type
            assert dtype_actual == dtype_expected

        return ch_list_renamed

    def delete_channels(self, ch_list: list[sy.Channel]) -> None:
        self._log_message("Deleting channels")
        self.console.channels.delete(ch_list)

        for ch in ch_list:
            exists, _ = self.console.channels.existing_channel(ch)
            assert not exists
