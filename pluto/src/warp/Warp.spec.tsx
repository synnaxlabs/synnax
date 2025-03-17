// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, it } from "vitest";

import { useRetrieve } from "@/warp/Warp";

describe("Warp", () => {
  it("should retrieve a value", () => {
    const { result } = renderHook(() =>
      useRetrieve({
        retrieve: async () => "test",
        retrieveChannels: async () => [],
        decode: async (value) => [value, true],
        queryKey: ["test"],
      }),
    );
  });
});
