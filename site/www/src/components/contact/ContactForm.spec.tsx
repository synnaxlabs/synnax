// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import { ContactForm } from "@/components/contact/ContactForm";

const ENDPOINT = "https://formspree.io/f/mgegebvq";
const FAILURE = "Something went wrong. Please try again later.";

const FIELDS = {
  name: "Gaal Dornik",
  email: "gaal@streeling.edu",
  phone: "555-555-5555",
  message: "We fire engines on a test stand.",
};

let fetch: Mock;
let alert: Mock;

const fill = (values: Partial<typeof FIELDS> = FIELDS): void => {
  const inputs: Record<keyof typeof FIELDS, HTMLElement> = {
    name: screen.getByPlaceholderText("Gaal Dornik"),
    email: screen.getByPlaceholderText("gaal@streeling.edu"),
    phone: screen.getByPlaceholderText("555-555-5555"),
    message: document.querySelector("textarea")!,
  };
  for (const [key, value] of Object.entries(values))
    fireEvent.change(inputs[key as keyof typeof FIELDS], { target: { value } });
};

const submit = (): HTMLButtonElement => screen.getByText("Submit →").closest("button")!;

const disabled = (): boolean => submit().getAttribute("aria-disabled") === "true";

const send = async (): Promise<void> => {
  await act(async () => {
    fireEvent.click(submit());
  });
};

const container = (): Element => document.querySelector(".contact-form-container")!;

describe("ContactForm", () => {
  beforeEach(() => {
    fetch = vi.fn(async () => new Response("{}", { status: 200 }));
    alert = vi.fn();
    vi.stubGlobal("fetch", fetch);
    vi.stubGlobal("alert", alert);
  });

  describe("validation", () => {
    it("should flag every empty field and send nothing", async () => {
      render(<ContactForm />);
      await send();
      for (const message of [
        "Please enter your name",
        "Please enter a valid email",
        "Please enter a phone number",
        "Please enter a message",
      ])
        expect(screen.getByText(message)).toBeTruthy();
      expect(fetch).not.toHaveBeenCalled();
    });

    it("should reject an email without a domain", async () => {
      render(<ContactForm />);
      fill({ ...FIELDS, email: "gaal" });
      await send();
      expect(screen.getByText("Please enter a valid email")).toBeTruthy();
      expect(fetch).not.toHaveBeenCalled();
    });

    it("should reject a message over 50,000 characters", async () => {
      render(<ContactForm />);
      fill({ ...FIELDS, message: "a".repeat(50001) });
      await send();
      expect(screen.getByText("Message is too long")).toBeTruthy();
      expect(fetch).not.toHaveBeenCalled();
    });
  });

  describe("submission", () => {
    it("should post each field to Formspree", async () => {
      render(<ContactForm />);
      fill();
      await send();
      expect(fetch).toHaveBeenCalledOnce();
      const [url, init] = fetch.mock.calls[0] as [string, RequestInit];
      expect(url).toBe(ENDPOINT);
      expect(init.method).toBe("POST");
      expect(init.headers).toEqual({ Accept: "application/json" });
      expect(Object.fromEntries(init.body as FormData)).toEqual(FIELDS);
    });

    it("should disable the button while the request is in flight", async () => {
      let respond!: (response: Response) => void;
      fetch.mockReturnValue(new Promise<Response>((resolve) => (respond = resolve)));
      render(<ContactForm />);
      fill();
      await send();
      expect(disabled()).toBe(true);
      await act(async () => respond(new Response("{}", { status: 500 })));
      expect(disabled()).toBe(false);
    });

    it("should send the form once when clicked again in flight", async () => {
      fetch.mockReturnValue(new Promise<Response>(() => {}));
      render(<ContactForm />);
      fill();
      await send();
      await send();
      expect(fetch).toHaveBeenCalledOnce();
    });

    it("should thank the visitor half a second after a success", async () => {
      vi.useFakeTimers();
      render(<ContactForm />);
      fill();
      await send();
      expect(disabled()).toBe(true);
      expect(container().classList).not.toContain("success");
      await act(async () => {
        await vi.advanceTimersByTimeAsync(500);
      });
      expect(container().classList).toContain("success");
      vi.useRealTimers();
    });

    it("should return to the form on back", async () => {
      vi.useFakeTimers();
      render(<ContactForm />);
      fill();
      await send();
      await act(async () => {
        await vi.advanceTimersByTimeAsync(500);
      });
      fireEvent.click(screen.getByText("Back"));
      expect(container().classList).not.toContain("success");
      expect(disabled()).toBe(false);
      vi.useRealTimers();
    });
  });

  describe("failure", () => {
    it("should alert and allow a retry when Formspree refuses the form", async () => {
      fetch.mockResolvedValue(new Response("{}", { status: 422 }));
      render(<ContactForm />);
      fill();
      await send();
      expect(alert).toHaveBeenCalledExactlyOnceWith(FAILURE);
      expect(disabled()).toBe(false);
      expect(container().classList).not.toContain("success");
    });

    it("should alert and allow a retry when the request fails", async () => {
      fetch.mockRejectedValue(new TypeError("Failed to fetch"));
      render(<ContactForm />);
      fill();
      await send();
      expect(alert).toHaveBeenCalledExactlyOnceWith(FAILURE);
      expect(disabled()).toBe(false);
    });
  });
});
