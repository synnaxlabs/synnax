// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Status } from "@/status";
import { status } from "@/status/aether";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { createAetherProvider } from "@/testutil/Aether";

const AetherProvider = createAetherProvider({
  ...synnax.REGISTRY,
  ...status.REGISTRY,
});

describe("Provider", () => {
  it("should keep its client when the error handler changes", () => {
    // A new log makes the aggregator hand out a new error handler.
    let log = (): void => {};
    const wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <AetherProvider>
        <Status.Aggregator log={log}>
          <Synnax.Provider connParams={TEST_CLIENT_PARAMS}>{children}</Synnax.Provider>
        </Status.Aggregator>
      </AetherProvider>
    );
    const { result, rerender, unmount } = renderHook(Synnax.use, { wrapper });
    const client = result.current;
    expect(client).not.toBeNull();
    log = () => {};
    rerender();
    expect(result.current).toBe(client);
    unmount();
  });
});
