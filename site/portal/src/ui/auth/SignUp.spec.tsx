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
import { cleanup, render, screen } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { type Invited } from "@/server/invitation";
import { SignUp } from "@/ui/auth/SignUp";

const INVITED = {
  invitation: { email: "ada@acme.co" },
  team: { name: "Acme" },
} as Invited;

const email = (): HTMLInputElement =>
  screen.getByDisplayValue<HTMLInputElement>("ada@acme.co");

describe("SignUp", () => {
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

  afterEach(cleanup);

  it("should keep an invited address fixed and visible to password managers", () => {
    render(<SignUp target="/" invited={INVITED} />);
    const input = email();
    expect(input.readOnly).toBe(true);
    expect(input.disabled).toBe(false);
    expect(input.autocomplete).toBe("username");
  });

  it("should tell the visitor that the account gets release notes", () => {
    render(<SignUp target="/" invited={null} />);
    expect(
      screen.getByText(
        "We email you our release notes. You can unsubscribe at any time.",
      ),
    ).toBeTruthy();
  });
});
