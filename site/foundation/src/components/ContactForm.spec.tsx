// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ContactForm } from "@/components/ContactForm";

const VALUES = {
  name: "Alex Rivera",
  email: "alex@example.com",
  company: "Example Engineering",
  message: "Connect our test stands across two facilities.",
};

const setup = (respond: () => Promise<Response> = async () => new Response("{}")) => {
  const fetch = vi.fn(respond);
  vi.stubGlobal("fetch", fetch);
  render(<ContactForm />);
  return fetch;
};

const fill = (): void => {
  fireEvent.change(screen.getByLabelText("Name"), { target: { value: VALUES.name } });
  fireEvent.change(screen.getByLabelText("Work email"), {
    target: { value: VALUES.email },
  });
  fireEvent.change(screen.getByLabelText("Company"), {
    target: { value: VALUES.company },
  });
  fireEvent.change(screen.getByLabelText("What are you working on?"), {
    target: { value: VALUES.message },
  });
};

const send = async (): Promise<void> => {
  await act(async () => {
    fireEvent.submit(screen.getByRole("form", { name: "Contact our team" }));
  });
};

describe("ContactForm", () => {
  afterEach(cleanup);

  it("should validate required fields and email before sending", async () => {
    const fetch = setup();
    await send();
    expect(fetch).not.toHaveBeenCalled();
    fill();
    fireEvent.change(screen.getByLabelText("Work email"), {
      target: { value: "not-an-email" },
    });
    await send();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("should send a complete request without requiring a phone and focus the acknowledgement", async () => {
    const fetch = setup();
    fill();
    await send();
    expect(fetch).toHaveBeenCalledOnce();
    const [endpoint, request] = vi.mocked(globalThis.fetch).mock.calls[0];
    expect(endpoint).toBe("https://formspree.io/f/mgegebvq");
    expect(request?.method).toBe("POST");
    expect(request?.headers).toEqual({ Accept: "application/json" });
    expect(Object.fromEntries(request?.body as FormData)).toEqual({
      ...VALUES,
      source: "Foundation",
    });
    const acknowledgement = screen.getByRole("status");
    expect(acknowledgement.textContent).toContain("Thanks for reaching out.");
    expect(document.activeElement).toBe(acknowledgement);
    fireEvent.click(screen.getByRole("button", { name: "Send another message" }));
    expect(screen.getByLabelText<HTMLInputElement>("Name").value).toBe("");
    expect(document.activeElement).toBe(screen.getByLabelText("Name"));
  });

  it("should lock the draft and prevent duplicate submissions while pending", async () => {
    const { promise, resolve } = Promise.withResolvers<Response>();
    const fetch = setup(() => promise);
    fill();
    await send();
    await send();
    expect(fetch).toHaveBeenCalledOnce();
    expect(screen.getByLabelText<HTMLInputElement>("Name").disabled).toBe(true);
    expect(
      screen.getByLabelText<HTMLTextAreaElement>("What are you working on?").disabled,
    ).toBe(true);
    expect(
      screen.getByRole("button", { name: "Sending" }).getAttribute("aria-disabled"),
    ).toBe("true");
    await act(async () => resolve(new Response("{}")));
    expect(screen.getByRole("status")).toBeTruthy();
  });

  it.each(["response", "network"])(
    "should preserve the draft after a %s failure and allow retry",
    async (failure) => {
      const fetch = setup(async () => {
        if (failure === "network") throw new TypeError("Failed to fetch");
        return new Response("{}", { status: 422 });
      });
      fill();
      fireEvent.change(screen.getByLabelText("Phone (optional)"), {
        target: { value: "+1 555 0100" },
      });
      await send();
      expect(screen.getByRole("alert").textContent).toContain("Please try again");
      expect(
        screen.getByLabelText<HTMLTextAreaElement>("What are you working on?").value,
      ).toBe(VALUES.message);
      expect(screen.getByLabelText<HTMLInputElement>("Name").disabled).toBe(false);
      expect(
        screen
          .getByRole("button", { name: "Talk to our team" })
          .getAttribute("aria-disabled"),
      ).toBeNull();
      fetch.mockResolvedValueOnce(new Response("{}"));
      await send();
      expect(fetch).toHaveBeenCalledTimes(2);
      const [, request] = vi.mocked(globalThis.fetch).mock.calls[1];
      expect(Object.fromEntries(request?.body as FormData)).toEqual({
        ...VALUES,
        phone: "+1 555 0100",
        source: "Foundation",
      });
      expect(screen.queryByRole("alert")).toBeNull();
      expect(screen.getByRole("status")).toBeTruthy();
    },
  );
});
