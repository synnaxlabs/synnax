#  Copyright 2023 Synnax Labs, Inc.
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
def populate(num_channels, num_ranges, num_samples):
    client = instantiate_client()

    for channel_index in range(num_channels):
        channel_name = generate_channel_name(channel_index)
        client.create_channel(channel_name)

        for range_index in range(num_ranges):
            range_data = generate_fake_data(num_samples)
            client.populate_range(channel_name, range_data)


def instantiate_client():
    return MockClient()


def generate_channel_name(channel_index):
    return f"Channel_{channel_index}"


def generate_fake_data(num_samples):
    return [random.randint(0, 100) for _ in range(num_samples)]


class MockClient:
    def create_channel(self, channel_name):
        print(f"Creating channel: {channel_name}")

    def populate_range(self, channel_name, range_data):
        print(f"Populating range in channel {channel_name} with data: {range_data}")


if __name__ == "__main__":
    populate()
