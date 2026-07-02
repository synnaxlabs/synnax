// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Cluster } from "@/platform/cluster";
import { clusterState } from "@/platform/cluster/testutil";
import { renderHookWithConsole } from "@/testutil";

describe("useCopyLinkToClipboard", () => {
  const writeText = vi.fn(async (_text: string) => {});

  beforeEach(() => {
    writeText.mockClear();
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should copy a link containing the active cluster key and ontology id", async () => {
    const id: ontology.ID = { type: "channel", key: "42" };
    const { result } = await renderHookWithConsole(
      () => Cluster.useCopyLinkToClipboard(),
      { preloadedState: clusterState([], "cluster-1") },
    );
    await act(async () => {
      result.current({ name: "My Channel", ontologyID: id });
    });
    expect(writeText).toHaveBeenCalledTimes(1);
    const url = writeText.mock.calls[0][0];
    expect(url).toContain("cluster-1");
    expect(url).toContain("channel");
    expect(url).toContain("42");
  });

  it("should not copy anything when there is no active cluster", async () => {
    const id: ontology.ID = { type: "channel", key: "42" };
    const { result } = await renderHookWithConsole(
      () => Cluster.useCopyLinkToClipboard(),
      { preloadedState: clusterState([], undefined) },
    );
    await act(async () => {
      result.current({ name: "My Channel", ontologyID: id });
    });
    expect(writeText).not.toHaveBeenCalled();
  });
});
