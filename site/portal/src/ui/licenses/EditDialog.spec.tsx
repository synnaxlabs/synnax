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

import { type schema } from "@/server/db";
import { LICENSE, NOW } from "@/server/license/testutil";
import { Licenses } from "@/ui/licenses";

const ACME: schema.Organization = {
  key: LICENSE.organization,
  kind: "team",
  name: "Acme",
  clerkOrgID: "org_acme",
  ownerUserID: null,
  createdAt: NOW,
};

const openEdit = (license: Partial<schema.License> = {}): void => {
  render(
    <Licenses.License
      license={{ ...LICENSE, ...license }}
      organization={ACME}
      activations={[]}
      events={[]}
      actors={{}}
      staff
      now={NOW}
    />,
  );
  fireEvent.click(screen.getByText("Edit"));
};

describe("EditDialog", () => {
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
    vi.restoreAllMocks();
  });

  it("should fill the form with the license's current terms", async () => {
    openEdit({ label: "Bench rig", maxVersion: "0.62" });
    await waitFor(() => expect(screen.getByDisplayValue("Bench rig")).toBeTruthy());
    expect(screen.getByDisplayValue("2027-03-01")).toBeTruthy();
    expect(screen.getByDisplayValue("0.62")).toBeTruthy();
  });

  it("should reject a perpetual license with no maximum version", async () => {
    const fetch = vi.spyOn(globalThis, "fetch");
    openEdit({ term: "perpetual", expiresAt: null });
    await waitFor(() => expect(screen.getByText("Save")).toBeTruthy());
    fireEvent.click(screen.getByText("Save"));
    await waitFor(() =>
      expect(
        screen.getByText("A perpetual license needs a maximum version"),
      ).toBeTruthy(),
    );
    expect(fetch).not.toHaveBeenCalled();
  });
});
