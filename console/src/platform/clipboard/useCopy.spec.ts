// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { Clipboard } from "@/platform/clipboard";
import {
  renderHookWithConsole,
  stubClipboardUnavailable,
  stubClipboardWriteText,
  stubCopyCommand,
  stubCopyCommandThrowing,
  stubCopyCommandUnavailable,
} from "@/testutil";

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

  // An insecure origin denies the clipboard API, so the copy command carries the text.
  it("copies through the command when the clipboard API denies the write", async () => {
    writeText = stubClipboardWriteText(async () => {
      throw new Error("denied");
    });
    const copied = stubCopyCommand();
    const { result } = await renderCopy();
    act(() => result.current.copy("hello", "greeting"));
    await waitFor(() =>
      expect(
        hasStatus(
          result.current.notifications,
          "success",
          "Copied greeting to clipboard",
        ),
      ).toBe(true),
    );
    expect(copied).toHaveBeenCalledWith("hello");
  });

  it("copies through the command when the clipboard API is absent", async () => {
    stubClipboardUnavailable();
    const copied = stubCopyCommand();
    const { result } = await renderCopy();
    act(() => result.current.copy("hello", "greeting"));
    await waitFor(() => expect(copied).toHaveBeenCalledWith("hello"));
  });

  it("leaves no scratch element behind after copying through the command", async () => {
    stubClipboardUnavailable();
    const copied = stubCopyCommand();
    const { result } = await renderCopy();
    act(() => result.current.copy("hello", "greeting"));
    await waitFor(() => expect(copied).toHaveBeenCalled());
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("cleans up and reports an error when the copy command throws", async () => {
    stubClipboardUnavailable();
    stubCopyCommandThrowing();
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
    expect(document.querySelector("textarea")).toBeNull();
  });

  // Pins the behavior for the day an engine drops the deprecated command.
  it("reports an error status when the copy command is gone", async () => {
    stubClipboardUnavailable();
    stubCopyCommandUnavailable();
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
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("reports an error status when the command also fails", async () => {
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
