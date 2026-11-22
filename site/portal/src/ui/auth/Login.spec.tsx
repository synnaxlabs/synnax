// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment jsdom

import { ResizeObserver } from "@juggle/resize-observer";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { Invited } from "@/server/invitation";
import { Auth } from "@/ui/auth";

// Clerk is the external service the portal signs in through. Its stores hand Login a
// client that answers each sign-in step as the test asks.
const clerk = vi.hoisted(() => ({
  client: { signIn: { create: vi.fn() } },
}));

vi.mock("@clerk/astro/client", () => {
  const store = <T,>(value: () => T) => ({
    subscribe: (listener: (value: T) => void) => {
      listener(value());
      return () => {};
    },
  });
  return {
    $isLoadedStore: store(() => true),
    $clerkStore: store(() => clerk),
    $userStore: store(() => null),
  };
});

const EMAIL_CODE = {
  strategy: "email_code",
  emailAddressId: "idn_email",
  safeIdentifier: "a•••@acme.co",
};
const BACKUP_CODE = { strategy: "backup_code" };

const INVITED = {
  invitation: { email: "ada@acme.co" },
  team: { name: "Acme" },
} as Invited;

const email = (): HTMLInputElement =>
  screen.getByDisplayValue<HTMLInputElement>("ada@acme.co");

describe("Login", () => {
  // jsdom lacks the ResizeObserver, matchMedia, and canvas that Lyra's background uses.
  beforeAll(() => {
    vi.stubGlobal("ResizeObserver", ResizeObserver);
    vi.stubGlobal("matchMedia", (media: string): MediaQueryList =>
      Object.assign(new EventTarget(), {
        media,
        matches: false,
        onchange: null,
        addListener: () => {},
        removeListener: () => {},
      }),
    );
    HTMLCanvasElement.prototype.getContext = () => null;
  });

  afterAll(() => {
    vi.unstubAllGlobals();
  });

  afterEach(() => {
    cleanup();
    clerk.client.signIn.create.mockReset();
  });

  it("should keep an invited address fixed and visible to password managers", () => {
    render(<Auth.Login target="/" invited={INVITED} />);
    const input = email();
    expect(input.readOnly).toBe(true);
    expect(input.disabled).toBe(false);
    expect(input.autocomplete).toBe("username");
  });

  it("should tell a visitor at login that accounts get release notes", () => {
    render(<Auth.Login target="/" invited={null} />);
    expect(
      screen.getByText(
        "We email you our release notes. You can unsubscribe at any time.",
      ),
    ).toBeTruthy();
  });

  describe("second step", () => {
    const logIn = async (): Promise<ReturnType<typeof vi.fn>> => {
      const prepareSecondFactor = vi.fn(async () => ({}));
      clerk.client.signIn.create.mockResolvedValue({
        status: "needs_second_factor",
        supportedSecondFactors: [BACKUP_CODE, EMAIL_CODE],
        prepareSecondFactor,
      });
      render(<Auth.Login target="/" invited={INVITED} />);
      fireEvent.change(screen.getByLabelText("Password"), {
        target: { value: "hunter2" },
      });
      fireEvent.click(screen.getByText("Log in", { selector: "button" }));
      await waitFor(() =>
        expect(screen.getByText("Two-step verification")).toBeTruthy(),
      );
      return prepareSecondFactor;
    };

    it("should send the emailed code and say where it went", async () => {
      const prepare = await logIn();
      expect(prepare).toHaveBeenCalledWith({
        strategy: "email_code",
        emailAddressId: "idn_email",
      });
      expect(screen.getByText("Enter the code we sent to a•••@acme.co")).toBeTruthy();
    });

    it("should switch to a backup code when the user asks", async () => {
      await logIn();
      fireEvent.click(screen.getByText("Use a backup code"));
      expect(screen.getByText("Enter one of your backup codes")).toBeTruthy();
      expect(screen.queryByText("Use a backup code")).toBeNull();
    });
  });
});
