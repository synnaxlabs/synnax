// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { License } from "@/license";
import { Synnax } from "@/synnax";
import { createSynnaxWrapper } from "@/testutil/Synnax";

/** Renders the license query under a client that a spec can swap. */
const renderResult = async (client: Client) => {
  const held = { client };
  const Outer = createSynnaxWrapper({ client: null });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Outer>
      <Synnax.TestProvider client={held.client}>{children}</Synnax.TestProvider>
    </Outer>
  );
  Wrapper.displayName = "SwappableClientWrapper";
  const { result, rerender } = renderHook(() => License.useResult({}), {
    wrapper: Wrapper,
  });
  await waitFor(() => expect(result.current.variant).toEqual("success"));
  return { result, rerender, held };
};

describe("License queries", () => {
  describe("useResult", () => {
    it("should retrieve the Core's license", async () => {
      const client = createTestClient();
      const { result } = await renderResult(client);
      expect(result.current.data).toEqual(await client.license.retrieve());
    });

    it("should drop the previous Core's license as soon as the Core changes", async () => {
      const { result, rerender, held } = await renderResult(createTestClient());
      held.client = createTestClient();
      rerender();
      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toBeUndefined();
      await waitFor(() => expect(result.current.variant).toEqual("success"));
    });
  });
});
