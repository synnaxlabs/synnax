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
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";

import { NOW } from "@/server/license/testutil";
import { Admin } from "@/ui/admin";

const chooseTerm = (current: string, next: string): void => {
  fireEvent.click(screen.getByText(current));
  fireEvent.click(screen.getByText(next));
};

describe("Admin.Licenses", () => {
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

  beforeEach(() => {
    render(
      <Admin.Licenses
        teams={[{ clerkOrgID: "org_acme", name: "Acme" }]}
        licenses={[]}
        now={NOW}
      />,
    );
    fireEvent.click(screen.getByText("Issue a license"));
  });

  afterEach(() => {
    cleanup();
  });

  describe("issue dialog", () => {
    it("should ask a subscription for its expiry only", () => {
      expect(screen.queryByText("Expires (UTC)")).not.toBeNull();
      expect(screen.queryByText("Maximum version")).toBeNull();
    });

    it("should ask perpetual with updates for an end date and a version", () => {
      chooseTerm("Subscription", "Perpetual with updates");
      expect(screen.queryByText("Updates until (UTC)")).not.toBeNull();
      expect(screen.queryByText("Maximum version")).not.toBeNull();
    });

    it("should ask a perpetual license for its maximum version only", () => {
      chooseTerm("Subscription", "Perpetual");
      expect(screen.queryByText("Expires (UTC)")).toBeNull();
      expect(screen.queryByText("Maximum version")).not.toBeNull();
    });

    it("should ask for a channel count once the channel limit is checked", () => {
      expect(screen.queryByText("Channels per Core")).toBeNull();
      fireEvent.click(screen.getByText("Limit channels per Core"));
      expect(screen.queryByText("Channels per Core")).not.toBeNull();
    });
  });
});
