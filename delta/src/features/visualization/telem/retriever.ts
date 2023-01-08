/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

import { Frame, Synnax, TimeRange } from "@synnaxlabs/client";

import { Range } from "@/features/workspace";

export class FrameRetriever {
  private readonly client: Synnax;

  constructor(client: Synnax) {
    this.client = client;
  }

  async get(req: FrameRetrieverRequest): Promise<Frame> {
    const { range, keys } = req;
    const tr = new TimeRange(range.start, range.end);
    return await this.client.data.readFrame(tr, keys);
  }
}

export interface FrameRetrieverRequest {
  range: Range;
  keys: string[];
}
