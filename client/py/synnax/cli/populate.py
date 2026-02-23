#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import click


@click.command()
@click.option(
    "--num-channels", type=int, default=50, help="Number of channels to create"
)
@click.option("--num-ranges", type=int, default=20, help="Number of ranges to create")
@click.option(
    "--num-samples",
    type=int,
    default=10,
    help="Number of samples per range in each channel",
)
def populate(num_channels: int, num_ranges: int, num_samples: int) -> None:
    client = instantiate_client()

    for channel_index in range(num_channels):
        channel_name = generate_channel_name(channel_index)
        client.create_channel(channel_name)

        for range_index in range(num_ranges):
            range_data = generate_fake_data(num_samples)
            client.populate_range(channel_name, range_data)


def instantiate_client() -> MockClient:
    return MockClient()


def generate_channel_name(channel_index: int) -> str:
    return f"Channel_{channel_index}"


def generate_fake_data(num_samples: int) -> list[int]:
    return [random.randint(0, 100) for _ in range(num_samples)]


class MockClient:
    def create_channel(self, channel_name: str) -> None:
        print(f"Creating channel: {channel_name}")

    def populate_range(self, channel_name: str, range_data: list[int]) -> None:
        print(f"Populating range in channel {channel_name} with data: {range_data}")


if __name__ == "__main__":
    populate()
