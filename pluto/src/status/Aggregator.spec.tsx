// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status as clientStatus } from "@synnaxlabs/client";
import { Status as Base } from "@synnaxlabs/lyra/status";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { Status } from "@/status";
import { status } from "@/status/aether";

const mount = async () => {
  const [workerSide, mainSide] = aether.createMockPair();
  const root = aether.render({ worker: workerSide, registry: status.REGISTRY });
  const wrapper = ({ children }: PropsWithChildren) => (
    <Aether.Provider worker={mainSide}>
      <Status.Aggregator>{children}</Status.Aggregator>
    </Aether.Provider>
  );
  const { result } = renderHook(() => Base.useNotifications(), { wrapper });
  await waitFor(() => expect(root.children).toHaveLength(1));
  const worker = root.children[0] as status.Aggregator;
  const report = (spec: clientStatus.Crude) =>
    act(() => worker.setState({ statuses: [clientStatus.create(spec)] }));
  const messages = () => result.current.statuses.map((s) => s.message);
  return { report, messages };
};

describe("Status.Aggregator", () => {
  it("should replace a status with a worker status that has the same key", async () => {
    const { report, messages } = await mount();
    report({ key: "gl", variant: "error", message: "WebGL context lost" });
    await waitFor(() => expect(messages()).toEqual(["WebGL context lost"]));
    report({ key: "gl", variant: "success", message: "WebGL context restored" });
    await waitFor(() => expect(messages()).toEqual(["WebGL context restored"]));
  });
});
