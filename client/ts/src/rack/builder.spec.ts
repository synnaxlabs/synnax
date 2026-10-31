// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { filter } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { rack } from "@/rack";

describe("builder", () => {
  it("should build a request that the request schema accepts", () => {
    const request = filter.toRequest([
      rack.integrations.has("ni"),
      rack.hostIsNode.eq(true),
      rack.embedded.eq(false),
    ]);
    expect(rack.retrieveRequestZ.parse(request)).toEqual({
      where: {
        and: [
          { integrations: { any: ["ni"] } },
          { hostIsNode: true },
          { embedded: { eq: false } },
        ],
      },
    });
  });
});
