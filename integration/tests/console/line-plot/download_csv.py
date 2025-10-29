#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io
from console.case import ConsoleCase
import synnax as sy
import csv


class DownloadCSV(ConsoleCase):
    """
    Test the download CSV functionality of the plot.
    """

    def setup(self) -> None:
        super().setup()
        self.configure(loop_rate=0.5, manual_timeout=30)

    def run(self) -> None:
        console = self.console
        client = self.client
        index_channel = client.channels.create(
            name="download_csv_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        self.log(
            f"Created index channel {index_channel.name} with key {index_channel.key} and data type {index_channel.data_type}"
        )
        data_channel = client.channels.create(
            name="download_csv_data",
            data_type=sy.DataType.FLOAT32,
            index=index_channel.key,
        )
        self.log(
            f"Created data channel {data_channel.name} with key {data_channel.key} and data type {data_channel.data_type}"
        )
        end = sy.TimeStamp.now()
        start = end - sy.TimeSpan.MINUTE
        time_data: list[sy.TimeStamp] = []
        data_data: list[float] = []
        time_i = start
        i = 0
        while time_i.before_eq(end):
            time_data.append(time_i)
            data_data.append(i)
            time_i += sy.TimeSpan.SECOND
            i += 1

        client.write(
            start,
            {
                index_channel.key: time_data,
                data_channel.key: data_data,
            },
        )
        console.plot.new()
        console.plot.add_channels("Y1", [index_channel.name, data_channel.name])
        console.plot.add_ranges(["30m"])
        csv_data = console.plot.download_csv()
        csv_file = io.StringIO(csv_data)
        reader = csv.reader(csv_file)
        header = next(reader)
        assert header[0] == index_channel.name
        assert header[1] == data_channel.name
        i = 0
        for row in reader:
            assert row[0] == str(
                int(time_data[i])
            ), f"Row {i} has incorrect index: {row[0]} != {str(int(time_data[i]))}"
            assert row[1] == str(
                data_data[i]
            ), f"Row {i} has incorrect data: {row[1]} != {str(data_data[i])}"
            i += 1
