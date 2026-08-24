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
import { afterEach, beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { Core } from "@/platform/core";
import { createCore, createCoreState } from "@/session/core/testutil";
import { renderHookWithConsole, stubClipboardWriteText } from "@/testutil";

const CORE = createCore("Alpha", { clusterKey: "cluster-1" });

describe("useCopyLinkToClipboard", () => {
  let writeText: Mock;

  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should copy a link containing the active cluster key and ontology id", async () => {
    const id: ontology.ID = { type: "channel", key: "42" };
    const { result } = await renderHookWithConsole(
      () => Core.useCopyLinkToClipboard(),
      { preloadedState: createCoreState([CORE], CORE.key) },
    );
    await act(async () => {
      result.current({ name: "My Channel", ontologyID: id });
    });
    expect(writeText).toHaveBeenCalledTimes(1);
    expect(writeText.mock.calls[0][0]).toBe("synnax://cluster/cluster-1/channel/42");
  });

  it("should not copy anything when there is no active Core", async () => {
    const id: ontology.ID = { type: "channel", key: "42" };
    const { result } = await renderHookWithConsole(
      () => Core.useCopyLinkToClipboard(),
      { preloadedState: createCoreState([], undefined) },
    );
    await act(async () => {
      result.current({ name: "My Channel", ontologyID: id });
    });
    expect(writeText).not.toHaveBeenCalled();
  });
});
