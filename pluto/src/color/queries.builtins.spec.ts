// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { color as xcolor } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Color } from "@/color";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const remote = createTestClient();

describe("queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    await remote.connect();
  });

  describe("useRestore", () => {
    it("should write the shipped values back onto an edited built-in", async () => {
      const shipped = (await client.colors.retrieveBuiltins()).at(-1);
      if (shipped == null) throw new Error("the Core ships no named colors");
      await remote.colors.retrieve(shipped.key);
      await remote.colors.dispatch(
        shipped.key,
        color.set({ path: ["light"], value: xcolor.construct("#123456") }),
      );
      const { result } = renderHook(Color.useRestore, { wrapper });
      await act(async () => {
        await result.current.updateAsync(shipped);
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().colors.retrieve(shipped.key);
      expect(stored.light).toEqual(shipped.light);
      expect(stored.dark).toEqual(shipped.dark);
    });
  });
});
