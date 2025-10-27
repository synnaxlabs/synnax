// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { View } from "@/view";

const client = createTestClient();

describe("View queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should retrieve a list of views", async () => {
      const view1 = await client.views.create({
        name: "View 1",
        type: "lineplot",
        query: { channels: ["ch1"] },
      });
      const view2 = await client.views.create({
        name: "View 2",
        type: "table",
        query: { channels: ["ch2"] },
      });

      const { result } = renderHook(() => View.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(view1.key);
      expect(result.current.data).toContain(view2.key);
    });
  });
});
