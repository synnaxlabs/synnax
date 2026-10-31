// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, filter } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";

describe("builder", () => {
  it("should build a request that the request schema accepts", () => {
    const request = filter.toRequest([
      channel.virtual.eq(false),
      channel.isIndex.eq(true),
      channel.dataType.eq(DataType.TIMESTAMP),
      channel.namePatterns.hasAny(["pressure_.*"]),
      channel.search("pressure"),
    ]);
    expect(channel.retrieveRequestZ.parse(request)).toEqual({
      where: {
        and: [
          { virtual: false },
          { isIndex: { eq: true } },
          { dataType: { eq: DataType.TIMESTAMP } },
          { namePatterns: ["pressure_.*"] },
        ],
      },
      searchTerm: "pressure",
    });
  });
});
