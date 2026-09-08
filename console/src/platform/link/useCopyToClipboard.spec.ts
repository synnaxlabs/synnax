// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { Link } from "@/platform/link";
import { renderHookWithConsole, stubClipboardWriteText } from "@/testutil";

describe("Link.useCopyToClipboard", () => {
  let writeText: Mock;
  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("copies a cluster link when no ontology ID is given", async () => {
    const { result } = await renderHookWithConsole(() => Link.useCopyToClipboard());
    act(() => result.current({ clusterKey: "abc", name: "My Core" }));
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

  // A Core caches its cluster key on the first connection, so a link to one never
  // connected would name nothing the person receiving it could open.
  it("copies nothing for a Core that has never connected", async () => {
    const { result } = await renderHookWithConsole(() => ({
      copy: Link.useCopyToClipboard(),
      notifications: Status.useNotifications(),
    }));
    act(() => result.current.copy({ clusterKey: undefined, name: "My Core" }));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.map(({ variant, message }) => ({
          variant,
          message,
        })),
      ).toEqual([{ variant: "error", message: "Failed to copy link to My Core" }]),
    );
    expect(writeText).not.toHaveBeenCalled();
  });
});
