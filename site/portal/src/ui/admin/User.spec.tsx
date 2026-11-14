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
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  type Mock,
  vi,
} from "vitest";

import { type Organization } from "@/server/db/schema";
import { NOW } from "@/server/license/testutil";
import { NONE } from "@/server/usage";
import { Admin } from "@/ui/admin";

const ACCOUNT = {
  userID: "user_ada",
  name: "Ada Lovelace",
  email: "ada@example.com",
  joinedAt: NOW,
};

const personalOf = (desktopChannels: number | null): Organization => ({
  key: crypto.randomUUID(),
  kind: "personal",
  name: "Ada Lovelace",
  clerkOrgID: null,
  ownerUserID: ACCOUNT.userID,
  desktopChannels,
  createdAt: NOW,
});

describe("Admin.User", () => {
  let fetchMock: Mock;

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

  beforeEach(() => {
    fetchMock = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    cleanup();
  });

  afterAll(() => {
    vi.unstubAllGlobals();
  });

  const renderUser = (personal: Organization | null): void => {
    render(
      <Admin.User
        account={ACCOUNT}
        usage={NONE}
        personal={personal}
        machines={[]}
        teams={[]}
        activity={null}
        now={NOW}
      />,
    );
  };

  const posted = (): unknown => {
    const [path, init] = fetchMock.mock.calls[0] as [string, { body: string }];
    expect(path).toBe(`/api/users/${ACCOUNT.userID}/channels`);
    return JSON.parse(init.body);
  };

  const enter = (current: string, next: string): void => {
    const input = screen.getByDisplayValue(current);
    fireEvent.change(input, { target: { value: next } });
    fireEvent.blur(input);
  };

  const choose = (current: string, next: string): void => {
    fireEvent.click(screen.getByText(current));
    fireEvent.click(screen.getByText(next));
  };

  describe("Desktop channels", () => {
    it("should show the default cap of a user with none set", () => {
      renderUser(personalOf(null));
      expect(screen.queryByText("55 (default)")).not.toBeNull();
    });

    it("should show the cap set for a user", () => {
      renderUser(personalOf(0));
      expect(screen.queryByText("Unlimited")).not.toBeNull();
    });

    it("should offer no change before the user's first visit", () => {
      renderUser(null);
      expect(screen.queryByText("55 (default)")).not.toBeNull();
      expect(screen.queryByText("Desktop channels", { selector: "button" })).toBeNull();
    });

    it("should post a cap that staff enter", async () => {
      renderUser(personalOf(null));
      fireEvent.click(screen.getByRole("button", { name: /Desktop channels/ }));
      choose("Default (55)", "Limited");
      enter("55", "200");
      fireEvent.click(screen.getByText("Save"));
      await waitFor(() => expect(fetchMock).toHaveBeenCalled());
      expect(posted()).toEqual({ channels: "200" });
    });

    it("should post an empty cap for the default", async () => {
      renderUser(personalOf(200));
      fireEvent.click(screen.getByRole("button", { name: /Desktop channels/ }));
      choose("Limited", "Default (55)");
      fireEvent.click(screen.getByText("Save"));
      await waitFor(() => expect(fetchMock).toHaveBeenCalled());
      expect(posted()).toEqual({ channels: "" });
    });

    it("should post a cap of 0 for unlimited channels", async () => {
      renderUser(personalOf(null));
      fireEvent.click(screen.getByRole("button", { name: /Desktop channels/ }));
      choose("Default (55)", "Unlimited");
      fireEvent.click(screen.getByText("Save"));
      await waitFor(() => expect(fetchMock).toHaveBeenCalled());
      expect(posted()).toEqual({ channels: "0" });
    });

    it("should refuse a limited cap below 1", async () => {
      renderUser(personalOf(200));
      fireEvent.click(screen.getByRole("button", { name: /Desktop channels/ }));
      enter("200", "0");
      fireEvent.click(screen.getByText("Save"));
      expect(
        await screen.findByText("Use a whole number of at least 1"),
      ).not.toBeNull();
      expect(fetchMock).not.toHaveBeenCalled();
    });
  });
});
