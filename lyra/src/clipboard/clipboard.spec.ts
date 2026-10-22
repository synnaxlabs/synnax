// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { clipboard } from "@/clipboard";
import {
  stubClipboardUnavailable,
  stubClipboardWriteText,
  stubCopyCommand,
  stubCopyCommandThrowing,
  stubCopyCommandUnavailable,
} from "@/testutil";

describe("clipboard.writeText", () => {
  it("should write through the clipboard API when it is available", async () => {
    const writeText = stubClipboardWriteText();
    const copied = stubCopyCommand();
    await clipboard.writeText("hello");
    expect(writeText).toHaveBeenCalledWith("hello");
    expect(copied).not.toHaveBeenCalled();
  });

  it("should copy through the command when the clipboard API denies the write", async () => {
    stubClipboardWriteText(async () => {
      throw new Error("denied");
    });
    const copied = stubCopyCommand();
    await clipboard.writeText("hello");
    expect(copied).toHaveBeenCalledWith("hello");
  });

  it("should copy through the command when the clipboard API is absent", async () => {
    stubClipboardUnavailable();
    const copied = stubCopyCommand();
    await clipboard.writeText("hello");
    expect(copied).toHaveBeenCalledWith("hello");
  });

  it("should leave no scratch element behind after copying through the command", async () => {
    stubClipboardUnavailable();
    stubCopyCommand();
    await clipboard.writeText("hello");
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("should reject with the API's denial as the cause when both paths fail", async () => {
    const denial = new Error("denied");
    stubClipboardWriteText(async () => {
      throw denial;
    });
    stubCopyCommand(false);
    await expect(clipboard.writeText("hello")).rejects.toMatchObject({
      message: "Failed to copy to the clipboard",
      cause: denial,
    });
  });

  it("should clean up and reject when the command throws", async () => {
    stubClipboardUnavailable();
    stubCopyCommandThrowing();
    await expect(clipboard.writeText("hello")).rejects.toThrow("blocked");
    expect(document.querySelector("textarea")).toBeNull();
  });

  it("should reject when the command is gone", async () => {
    stubClipboardUnavailable();
    stubCopyCommandUnavailable();
    await expect(clipboard.writeText("hello")).rejects.toThrow(
      "Failed to copy to the clipboard",
    );
  });
});
