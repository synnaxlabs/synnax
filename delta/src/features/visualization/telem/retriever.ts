import { Frame, Synnax, TimeRange } from "@synnaxlabs/client";

import { Range } from "@/features/workspace";

export class FrameRetriever {
  private readonly client: Synnax;

  constructor(client: Synnax) {
    this.client = client;
  }

  async get(req: FrameRetrieverRequest): Promise<Frame> {
    const { range, keys } = req;
    console.log(range.start, range.end);
    const tr = new TimeRange(range.start, range.end);
    console.log(tr.toString());
    return this.client.data.readFrame(tr, keys);
  }
}

export interface FrameRetrieverRequest {
  range: Range;
  keys: string[];
}
