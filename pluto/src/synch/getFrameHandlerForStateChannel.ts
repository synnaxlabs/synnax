// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type framer } from "@synnaxlabs/client";
import { type z } from "zod";

import { type FrameHandler } from "@/synch/types";

export const getFrameHandlerForStateChannel =
  <Z extends z.ZodTypeAny>(
    channel: channel.Name,
    schema: Z,
    onStateReceived: (state: z.output<Z>) => void,
  ): FrameHandler =>
  (frame: framer.Frame): void =>
    frame.get(channel).parseJSON(schema).forEach(onStateReceived);
