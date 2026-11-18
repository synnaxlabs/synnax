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

import { LICENSE, NOW } from "@/server/license/testutil";
import { Admin } from "@/ui/admin";

// jsdom cannot navigate, so the spec reads where a page load would have gone.
const { navigate } = vi.hoisted(() => ({ navigate: vi.fn<(url: string) => void>() }));
vi.mock("@/ui/api", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  navigate,
}));

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

  afterEach(() => {
    cleanup();
  });

  describe("list", () => {
    const entry = {
      license: LICENSE,
      organization: {
        key: LICENSE.organization,
        kind: "team" as const,
        name: "Acme",
        clerkOrgID: "org_acme",
        ownerUserID: null,
        desktopChannels: null,
        createdAt: NOW,
      },
      seats: 0,
    };

    it("should show the total and link the next page", () => {
      render(
        <Admin.Licenses
          teams={[]}
          page={{ entries: [entry], total: 51, next: { key: LICENSE.key } }}
          query={{ status: "active" }}
          nextHref={`/admin/licenses?status=active&after=${LICENSE.key}`}
          now={NOW}
        />,
      );
      expect(screen.queryByText("51 licenses")).not.toBeNull();
      expect(screen.getByText("Next page").closest("a")?.getAttribute("href")).toBe(
        `/admin/licenses?status=active&after=${LICENSE.key}`,
      );
      expect(screen.queryByText("First page")).toBeNull();
    });

    it("should link back to the first page, keeping the filter", () => {
      render(
        <Admin.Licenses
          teams={[]}
          page={{ entries: [entry], total: 51, next: null }}
          query={{ search: "rig", after: { key: LICENSE.key } }}
          nextHref={null}
          now={NOW}
        />,
      );
      expect(screen.getByText("First page").closest("a")?.getAttribute("href")).toBe(
        "/admin/licenses?search=rig",
      );
      expect(screen.queryByText("Next page")).toBeNull();
    });

    it("should say when nothing matches the filter", () => {
      render(
        <Admin.Licenses
          teams={[]}
          page={{ entries: [], total: 0, next: null }}
          query={{ status: "revoked" }}
          nextHref={null}
          now={NOW}
        />,
      );
      expect(screen.queryByText("No licenses match")).not.toBeNull();
      expect(screen.queryByText("0 licenses")).not.toBeNull();
    });

    describe("navigation", () => {
      beforeEach(() => {
        navigate.mockClear();
        render(
          <Admin.Licenses
            teams={[]}
            page={{ entries: [], total: 51, next: null }}
            query={{ search: "rig", status: "active", after: { key: LICENSE.key } }}
            nextHref={null}
            now={NOW}
          />,
        );
      });

      const type = (text: string): HTMLElement => {
        const input = screen.getByDisplayValue("rig");
        fireEvent.change(input, { target: { value: text } });
        return input;
      };

      it("should search on Enter from the first page, keeping the status", () => {
        fireEvent.keyDown(type(" bench "), { key: "Enter" });
        expect(navigate).toHaveBeenCalledWith(
          "/admin/licenses?search=bench&status=active",
        );
      });

      it("should not search before Enter", () => {
        type("bench");
        expect(navigate).not.toHaveBeenCalled();
      });

      it("should filter by a status with the search typed so far", () => {
        type("bench");
        fireEvent.click(screen.getByText("Revoked"));
        expect(navigate).toHaveBeenCalledWith(
          "/admin/licenses?search=bench&status=revoked",
        );
      });

      it("should clear the status when it is clicked again", () => {
        fireEvent.click(screen.getByText("Active"));
        expect(navigate).toHaveBeenCalledWith("/admin/licenses?search=rig");
      });
    });
  });

  describe("issue dialog", () => {
    beforeEach(() => {
      render(
        <Admin.Licenses
          teams={[{ clerkOrgID: "org_acme", name: "Acme" }]}
          page={{ entries: [], total: 0, next: null }}
          query={{}}
          nextHref={null}
          now={NOW}
        />,
      );
      fireEvent.click(screen.getByText("Issue a license"));
    });

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
