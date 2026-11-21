// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { enumeration } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Enumeration } from "@/enumeration";
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
    it("should write the shipped entries back onto an edited built-in", async () => {
      const shipped = (await client.enumerations.retrieveBuiltins()).at(-1);
      if (shipped == null) throw new Error("the Core ships no enumerations");
      await remote.enumerations.retrieve(shipped.key);
      await remote.enumerations.dispatch(shipped.key, [
        enumeration.set({ path: ["name"], value: `edited-${id.create()}` }),
        enumeration.set({ path: ["entries"], value: [] }),
      ]);
      const { result } = renderHook(Enumeration.useRestore, { wrapper });
      await act(async () => {
        await result.current.updateAsync(shipped);
      });
      expect(result.current.variant).toEqual("success");
      const stored = await createTestClient().enumerations.retrieve(shipped.key);
      expect(stored.name).toBe(shipped.name);
      expect(stored.entries).toEqual(shipped.entries);
    });
  });
});
