// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { stubClipboardWriteText, stubCopyCommand } from "@synnaxlabs/lyra/testutil";
import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { Clipboard } from "@/platform/clipboard";
import { renderHookWithConsole } from "@/testutil";

const renderCopy = async () =>
  await renderHookWithConsole(() => ({
    copy: Clipboard.useCopy(),
    notifications: Status.useNotifications(),
  }));

const hasStatus = (
  { statuses }: ReturnType<typeof Status.useNotifications>,
  variant: status.Variant,
  message: string,
): boolean => statuses.some((s) => s.variant === variant && s.message === message);

describe("Clipboard.useCopy", () => {
  let writeText: Mock;
  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("writes the text and reports a success status", async () => {
    const { result } = await renderCopy();
    act(() => result.current.copy("hello", "greeting"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("hello"));
    await waitFor(() =>
      expect(
        hasStatus(
          result.current.notifications,
          "success",
          "Copied greeting to clipboard",
        ),
      ).toBe(true),
    );
  });

  it("reports an error status when the write fails", async () => {
    writeText = stubClipboardWriteText(async () => {
      throw new Error("denied");
    });
    stubCopyCommand(false);
    const { result } = await renderCopy();
    act(() => result.current.copy("hello", "greeting"));
    await waitFor(() =>
      expect(
        hasStatus(
          result.current.notifications,
          "error",
          "Failed to copy greeting to clipboard",
        ),
      ).toBe(true),
    );
  });
});
