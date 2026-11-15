#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.case import ConsoleCase
from framework.utils import assert_link_format
from x import random_name


class ChannelContextMenu(ConsoleCase):
    """Test channel context menu operations."""

    suffix: str
    shared_index: str
    shared_data: str

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()
        self._create_shared_channels()

    def _create_shared_channels(self) -> None:
        """Create shared index + data channel for read-only tests."""
        self.shared_index = f"shared_idx_{self.suffix}"
        self.shared_data = f"shared_data_{self.suffix}"

        self.client.channels.create(
            name=self.shared_index,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        self.console.channels.wait_for_channels(self.shared_index, timeout=5.0)
        self.console.channels.create(
            name=self.shared_data,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )

    def teardown(self) -> None:
        with self._try_to("delete channels"):
            self.console.channels.delete([self.shared_data, self.shared_index])
        super().teardown()

    def run(self) -> None:
        """Run all channel context menu tests."""
        self.test_rename_channel()
        self.test_delete_channel()
        self.test_copy_link()

    def test_rename_channel(self) -> None:
        """Test renaming a channel via context menu."""
        self.log("Testing rename channel")

        console = self.console

        suffix = random_name()
        data_name = f"rename_data_{suffix}"
        new_name = f"renamed_data_{suffix}"

        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=self.shared_index,
        )

        console.channels.rename(names=data_name, new_names=new_name)

        ch = self.client.channels.retrieve(new_name)
        assert ch.name == new_name, f"Expected channel name {new_name}, got {ch.name}"

        console.channels.delete([new_name])

    def test_delete_channel(self) -> None:
        """Test deleting a channel via context menu."""
        self.log("Testing delete channel")

        console = self.console

        suffix = random_name()
        index_name = f"delete_idx_{suffix}"
        data_name = f"delete_data_{suffix}"

        console.channels.create(name=index_name, is_index=True)
        console.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_name,
        )

        console.channels.delete([data_name])
        console.channels.delete([index_name])

    def test_copy_link(self) -> None:
        """Test copying a channel link via context menu."""
        self.log("Testing copy channel link")

        link = self.console.channels.copy_link(self.shared_data)

        channel = self.client.channels.retrieve(self.shared_data)
        assert_link_format(link, resource_type="channel", resource_id=str(channel.key))
