// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AsyncTermSearcher, toArray } from "@synnaxlabs/x";

import { QueryError } from "@/errors";
import { FrameClient } from "@/framer";
import { RangeCreator } from "@/ranger/creator";
import {
  NewRangePayload,
  RangeKey,
  RangeKeys,
  RangeName,
  RangeNames,
  RangeParams,
  RangePayload,
  analyzeRangeParams,
} from "@/ranger/payload";
import { Range } from "@/ranger/range";
import { RangeRetriever } from "@/ranger/retriever";

export class RangeClient implements AsyncTermSearcher<string, RangeKey, Range> {
  private readonly frameClient: FrameClient;
  private readonly retriever: RangeRetriever;
  private readonly creator: RangeCreator;

  constructor(
    frameClient: FrameClient,
    retriever: RangeRetriever,
    creator: RangeCreator
  ) {
    this.frameClient = frameClient;
    this.retriever = retriever;
    this.creator = creator;
  }

  async create(range: NewRangePayload): Promise<Range>;

  async create(ranges: NewRangePayload[]): Promise<Range[]>;

  async create(ranges: NewRangePayload | NewRangePayload[]): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const res = this.sugar(await this.creator.create(toArray(ranges)));
    return single ? res[0] : res;
  }

  async search(term: string): Promise<Range[]> {
    return this.sugar(await this.retriever.search(term));
  }

  async retrieve(range: RangeKey | RangeName): Promise<Range>;

  async retrieve(params: RangeKeys | RangeNames): Promise<Range[]>;

  async retrieve(params: RangeParams): Promise<Range | Range[]> {
    const { single, actual } = analyzeRangeParams(params);
    const res = this.sugar(await this.retriever.retrieve(params));
    if (!single) return res;
    if (res.length === 0) throw new QueryError(`range matching ${actual} not found`);
    if (res.length > 1)
      throw new QueryError(`multiple ranges matching ${actual} found`);
    return res[0];
  }

  private sugar(payloads: RangePayload[]): Range[] {
    return payloads.map((payload) => {
      return new Range(payload.name, payload.timeRange, payload.key, this.frameClient);
    });
  }
}
