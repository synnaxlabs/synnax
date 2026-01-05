// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Variable } from "@/code/phantom";

export const GLOBALS: Variable[] = [
  {
    key: "set",
    name: "set",
    value: `
    --- Set the value of a channel
    --- @param channel string The name of the channel to set the value of
    --- @param value string The value to set the channel to
    --- @return void
    function(channel, value)
    end`,
  },
  {
    key: "elapsed_time",
    name: "elapsed_time",
    value: "0.0",
    docs: "The elapsed time of the sequence in seconds",
  },
  {
    key: "elapsed_time_within",
    name: "elapsed_time_within",
    value: `
    --- Check if the elapsed time of the sequence is within a given range.
    --- @param start the start time in seconds
    --- @param end the end time in seconds
    --- @return boolean true if the elapsed time is within the range, false otherwise
    function(start, end)
    end`,
  },
  {
    key: "iteration",
    name: "iteration",
    value: "0",
    docs: `The iteration number of the sequence. This is incremented each time the sequence is run,
    and starts at 1.`,
  },
  {
    key: "set_authority",
    name: "set_authority",
    value: `
    --- Set the control authority of a channel
    --- @param channel string The name of the channel to set the authority of
    --- @param authority number The authority to set the channel to
    --- @return void
    function(channel, authority)
    end`,
  },
];
