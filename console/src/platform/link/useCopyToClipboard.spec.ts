// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { stubClipboardWriteText } from "@/platform/clipboard/testutil";
import { Link } from "@/platform/link";
import { renderHookWithConsole } from "@/testutil";

describe("Link.useCopyToClipboard", () => {
  let writeText: Mock;
  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("copies a cluster link when no ontology ID is given", async () => {
    const { result } = await renderHookWithConsole(() => Link.useCopyToClipboard());
    act(() => result.current({ clusterKey: "abc", name: "My Cluster" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("synnax://cluster/abc"));
  });

  it("appends the ontology type and key when an ID is given", async () => {
    const { result } = await renderHookWithConsole(() => Link.useCopyToClipboard());
    act(() =>
      result.current({
        clusterKey: "abc",
        name: "My Range",
        ontologyID: { type: "range", key: "r1" },
      }),
    );
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith("synnax://cluster/abc/range/r1"),
    );
  });
});
