// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { FeedbackButton } from "@/components/feedback/Feedback";

describe("FeedbackButton", () => {
  let fetch: ReturnType<typeof vi.fn>;

  const open = (): HTMLButtonElement => {
    render(<FeedbackButton />);
    fireEvent.click(screen.getByText("Stuck? Let us know!"));
    return screen.getByText("Send").closest("button") as HTMLButtonElement;
  };

  const fill = (name: string, email: string, description: string): void => {
    fireEvent.change(screen.getByPlaceholderText("Gaal Dornik"), {
      target: { value: name },
    });
    fireEvent.change(screen.getByPlaceholderText("gaal@streeling.edu"), {
      target: { value: email },
    });
    fireEvent.change(screen.getByPlaceholderText("What can we improve?"), {
      target: { value: description },
    });
  };

  beforeEach(() => {
    fetch = vi.fn(async () => ({ ok: true }));
    vi.stubGlobal("fetch", fetch);
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe = vi.fn();
        unobserve = vi.fn();
        disconnect = vi.fn();
      },
    );
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  const disabled = (send: HTMLButtonElement): boolean =>
    send.getAttribute("aria-disabled") === "true";

  it("disables send until every field is filled", () => {
    const send = open();
    expect(disabled(send)).toBe(true);
    fill("Gaal", "gaal@streeling.edu", "   ");
    expect(disabled(send)).toBe(true);
    fireEvent.click(send);
    expect(fetch).not.toHaveBeenCalled();
    fill("Gaal", "gaal@streeling.edu", "Rename undone on deploy");
    expect(disabled(send)).toBe(false);
  });

  it("rejects an invalid email without posting", () => {
    const send = open();
    fill("Gaal", "not-an-email", "Rename undone on deploy");
    fireEvent.click(send);
    expect(screen.getByText("Enter a valid email")).toBeDefined();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("posts name, email, and description", async () => {
    const send = open();
    fill("Gaal", "gaal@streeling.edu", "Rename undone on deploy");
    fireEvent.click(send);
    await waitFor(() => expect(fetch).toHaveBeenCalledOnce());
    const body = fetch.mock.calls[0][1].body as FormData;
    expect(body.get("name")).toBe("Gaal");
    expect(body.get("email")).toBe("gaal@streeling.edu");
    expect(body.get("description")).toBe("Rename undone on deploy");
  });
});
