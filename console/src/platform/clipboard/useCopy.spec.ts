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

import { Clipboard } from "@/platform/clipboard";
import { stubClipboardWriteText } from "@/platform/clipboard/testutil";
import { renderHookWithConsole } from "@/testutil";

describe("Clipboard.useCopy", () => {
  let writeText: Mock;
  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("writes the text and reports a success status", async () => {
    const { result } = await renderHookWithConsole(() => ({
      copy: Clipboard.useCopy(),
      notifications: Status.useNotifications(),
    }));
    act(() => result.current.copy("hello", "greeting"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("hello"));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (s) =>
            s.variant === "success" && s.message === "Copied greeting to clipboard.",
        ),
      ).toBe(true),
    );
  });

  it("reports an error status when the write fails", async () => {
    writeText = stubClipboardWriteText(async () => {
      throw new Error("denied");
    });
    const { result } = await renderHookWithConsole(() => ({
      copy: Clipboard.useCopy(),
      notifications: Status.useNotifications(),
    }));
    act(() => result.current.copy("hello", "greeting"));
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (s) =>
            s.variant === "error" &&
            s.message === "Failed to copy greeting to clipboard",
        ),
      ).toBe(true),
    );
  });
});
