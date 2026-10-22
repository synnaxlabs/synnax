// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ContactForm } from "@/components/ContactForm";

const VALUES = {
  name: "Alex Rivera",
  email: "alex@example.com",
  company: "Example Engineering",
  message: "Connect our test stands across two facilities.",
};

const REQUIRED_LABELS = ["Name", "Work email", "Company", "What are you working on?"];

const expectFieldError = (field: HTMLElement): void => {
  expect(field.getAttribute("aria-invalid")).toBe("true");
  const description = field.getAttribute("aria-describedby");
  expect(description).toBeTruthy();
  expect(document.getElementById(description!)?.textContent?.trim()).toBeTruthy();
};

const setup = (respond: () => Promise<Response> = async () => new Response("{}")) => {
  const fetch = vi.fn(respond);
  vi.stubGlobal("fetch", fetch);
  render(<ContactForm />);
  for (const field of screen.getAllByRole("textbox")) field.scrollIntoView = vi.fn();
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

  it("should describe required field errors inline and focus the first invalid field", async () => {
    const fetch = setup();
    expect(
      screen.getByRole<HTMLFormElement>("form", { name: "Contact our team" })
        .noValidate,
    ).toBe(true);
    const name = screen.getByLabelText("Name");
    const scrollIntoView = vi.fn();
    name.scrollIntoView = scrollIntoView;
    await send();
    expect(fetch).not.toHaveBeenCalled();
    for (const label of REQUIRED_LABELS) expectFieldError(screen.getByLabelText(label));
    expect(document.activeElement).toBe(name);
    expect(scrollIntoView).toHaveBeenCalledExactlyOnceWith({
      block: "nearest",
      inline: "nearest",
      behavior: "instant",
    });
    expect(
      screen.getByLabelText("Phone (optional)").getAttribute("aria-invalid"),
    ).not.toBe("true");
  });

  it("should block a malformed email and focus its inline error", async () => {
    const fetch = setup();
    fill();
    const email = screen.getByLabelText("Work email");
    fireEvent.change(email, {
      target: { value: "not-an-email" },
    });
    await send();
    expect(fetch).not.toHaveBeenCalled();
    expectFieldError(email);
    expect(document.activeElement).toBe(email);
    expect(screen.getByLabelText("Name").getAttribute("aria-invalid")).not.toBe("true");
  });

  it("should reject whitespace-only required fields", async () => {
    const fetch = setup();
    for (const label of REQUIRED_LABELS)
      fireEvent.change(screen.getByLabelText(label), { target: { value: "   " } });
    await send();
    expect(fetch).not.toHaveBeenCalled();
    for (const label of REQUIRED_LABELS) expectFieldError(screen.getByLabelText(label));
  });

  it("should clear inline errors as fields are corrected and allow submission", async () => {
    const fetch = setup();
    await send();
    const descriptionIds = REQUIRED_LABELS.map((label) =>
      screen.getByLabelText(label).getAttribute("aria-describedby"),
    );
    fill();
    for (const label of REQUIRED_LABELS) {
      const field = screen.getByLabelText(label);
      expect(field.getAttribute("aria-invalid")).not.toBe("true");
      expect(field.getAttribute("aria-describedby")).toBeNull();
    }
    for (const id of descriptionIds) expect(document.getElementById(id!)).toBeNull();
    await send();
    expect(fetch).toHaveBeenCalledOnce();
    expect(screen.getByRole("status")).toBeTruthy();
  });

  it("should validate on blur and retain an error until the field is valid", () => {
    setup();
    const email = screen.getByLabelText("Work email");
    fireEvent.change(email, { target: { value: "not-an-email" } });
    expect(email.getAttribute("aria-invalid")).not.toBe("true");
    fireEvent.blur(email);
    expectFieldError(email);
    fireEvent.change(email, { target: { value: "still-invalid" } });
    expectFieldError(email);
    fireEvent.change(email, { target: { value: VALUES.email } });
    expect(email.getAttribute("aria-invalid")).not.toBe("true");
    expect(email.getAttribute("aria-describedby")).toBeNull();
  });

  it("should retain native required and email validation before hydration", () => {
    const page = new DOMParser().parseFromString(
      renderToStaticMarkup(<ContactForm />),
      "text/html",
    );
    const form = page.querySelector("form")!;
    expect(form.noValidate).toBe(false);
    for (const name of Object.keys(VALUES))
      expect(form.querySelector(`[name="${name}"]`)?.hasAttribute("required")).toBe(
        true,
      );
    expect(form.querySelector('[name="email"]')?.getAttribute("type")).toBe("email");
    expect(form.querySelector('[name="phone"]')?.hasAttribute("required")).toBe(false);
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
    expect(
      screen.getByRole<HTMLFormElement>("form", { name: "Contact our team" })
        .noValidate,
    ).toBe(true);
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
