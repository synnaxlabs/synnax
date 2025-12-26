// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * A simple example that creates a large number of channels that are indexed by a single
 * timestamp channel.
 */

import { DataType, Synnax } from "@synnaxlabs/client";

// Connect to a locally running, insecure Synnax cluster. If your connection parameters
// are different, enter them here. See https://docs.synnaxlabs.com/reference/client/quick-start.
const client = new Synnax({
  host: "localhost",
  port: 9090,
  username: "synnax",
  password: "seldon",
  secure: false,
});

const CHANNEL_COUNT = 100;

// Create an index channel that will be used to store our timestamps.
const timeChannel = await client.channels.create({
  name: "create_channels_time",
  dataType: DataType.TIMESTAMP,
  isIndex: true,
});

// Create data channels to store our data. Since we did not call client.channels.create
// here, the channels are not actually created in the Synnax cluster yet. We will do that
// in the next step.
let dataChannels = Array.from({ length: CHANNEL_COUNT }, (_, i) => ({
  name: `create_channels_data_${i}`,
  dataType: DataType.FLOAT64,
  index: timeChannel.key,
}));

// Notice how we reassign the result of the create call to the dataChannels variable.
// This means that all of the channels will have the correct key given to the channel by
// the server.
dataChannels = await client.channels.create(dataChannels);

client.close();
