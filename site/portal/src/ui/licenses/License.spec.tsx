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
import { uuid } from "@synnaxlabs/x";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { License, Organization } from "@/server/db/schema";
import { NOW } from "@/server/license/testutil";
import { Licenses } from "@/ui/licenses";

// jsdom cannot navigate, so the spec reads where a page load would have gone.
const { navigate } = vi.hoisted(() => ({ navigate: vi.fn<(url: string) => void>() }));
vi.mock("@/ui/api", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  navigate,
}));

const ACME: Organization = {
  key: uuid.create(),
  kind: "team",
  name: "Acme",
  clerkOrgID: "org_acme",
  ownerUserID: null,
  desktopChannels: null,
  createdAt: NOW,
};

const LICENSE: License = {
  key: uuid.create(),
  organization: ACME.key,
  edition: "enterprise",
  term: "subscription",
  nodes: 2,
  channels: 0,
  expiresAt: new Date("2027-03-01T00:00:00Z"),
  maxVersion: null,
  label: "Test rig",
  issuedBy: "user_staff",
  issuedAt: NOW,
  revokedAt: null,
};

const RENAMED = {
  event: {
    key: uuid.create(),
    at: NOW.toISOString(),
    kind: "amend",
    actor: "user_ada",
    organization: ACME.key,
    license: LICENSE.key,
    activation: null,
    detail: { label: { from: "Test rig", to: "Renamed rig" } },
  },
  actor: "Ada",
};

const renderLicense = (props: Partial<Licenses.LicenseProps> = {}): void => {
  render(
    <Licenses.License
      license={LICENSE}
      organization={ACME}
      activations={[]}
      events={[]}
      actors={{}}
      staff={false}
      admin
      now={NOW}
      {...props}
    />,
  );
};

const commit = async (label: string): Promise<void> => {
  fireEvent.click(screen.getByText("Rename"));
  const title = await waitFor(() => {
    const heading = screen.getByRole("heading", { name: LICENSE.label });
    expect(heading.getAttribute("contenteditable")).toBe("true");
    return heading;
  });
  title.innerText = label;
  await act(async () => {
    fireEvent.keyDown(title, { key: "Enter" });
  });
};

describe("Licenses.License", () => {
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

  it("should post the label an admin edits in place", async () => {
    const fetch = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation(async () => Response.json(RENAMED));
    renderLicense();
    await commit("Renamed rig");
    expect(await screen.findByText("License changed by Ada")).toBeTruthy();
    expect(fetch).toHaveBeenCalledWith(
      `/api/licenses/${LICENSE.key}/label`,
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ label: "Renamed rig" }),
      }),
    );
  });

  it("should keep the title plain until an admin starts a rename", () => {
    renderLicense();
    const title = screen.getByRole("heading", { name: LICENSE.label });
    fireEvent.doubleClick(title);
    expect(title.getAttribute("contenteditable")).toBeNull();
  });

  it("should return the title to plain text after a rename", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () =>
      Response.json(RENAMED),
    );
    renderLicense();
    await commit("Renamed rig");
    await waitFor(() =>
      expect(
        screen
          .getByRole("heading", { name: "Renamed rig" })
          .getAttribute("contenteditable"),
      ).toBeNull(),
    );
  });

  it("should keep dollar sequences in the tab title literal", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () =>
      Response.json(RENAMED),
    );
    document.title = `${LICENSE.label} | Synnax`;
    renderLicense();
    await commit("Rig $&");
    await waitFor(() => expect(document.title).toBe("Rig $& | Synnax"));
  });

  it("should refuse a second rename while one is saving", async () => {
    let finish = (_: Response): void => {};
    vi.spyOn(globalThis, "fetch").mockImplementation(
      async () => await new Promise((resolve) => (finish = resolve)),
    );
    renderLicense();
    await commit("Renamed rig");
    const button = screen.getByText("Rename").closest("button");
    await waitFor(() => expect(button?.getAttribute("aria-disabled")).toBe("true"));
    await act(async () => finish(Response.json(RENAMED)));
    await waitFor(() => expect(button?.getAttribute("aria-disabled")).toBeNull());
  });

  it("should show why a rename failed", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () =>
      Response.json(
        { error: "Only a team admin can rename a license" },
        { status: 403 },
      ),
    );
    renderLicense();
    await commit("Renamed rig");
    expect(
      await screen.findByText("Only a team admin can rename a license"),
    ).toBeTruthy();
    expect(screen.getByRole("heading", { name: LICENSE.label })).toBeTruthy();
  });

  it("should not offer a member a rename", () => {
    renderLicense({ admin: false });
    expect(screen.queryByText("Rename")).toBeNull();
    const title = screen.getByRole("heading", { name: LICENSE.label });
    expect(title.getAttribute("contenteditable")).toBeNull();
  });

  it("should not offer a rename on a Desktop license", () => {
    renderLicense({ license: { ...LICENSE, edition: "desktop" } });
    expect(screen.queryByText("Rename")).toBeNull();
  });

  describe("failed email", () => {
    const REVOKED = { ...LICENSE, revokedAt: NOW };

    afterEach(() => {
      navigate.mockReset();
      window.history.replaceState(null, "", "/");
    });

    it("should open the page that says so when a restore's email fails", async () => {
      window.history.replaceState(null, "", `/licenses/${LICENSE.key}?org=acme`);
      vi.spyOn(globalThis, "fetch").mockImplementation(async () =>
        Response.json(
          { error: '"Test rig" is restored, but the email about it failed.' },
          { status: 502 },
        ),
      );
      renderLicense({ license: REVOKED, staff: true });
      fireEvent.click(screen.getByText("Restore"));
      await waitFor(() => expect(navigate).toHaveBeenCalledOnce());
      const url = new URL(navigate.mock.calls[0][0]);
      expect(url.pathname).toBe(`/licenses/${LICENSE.key}`);
      expect(url.searchParams.get("org")).toBe("acme");
      expect(url.searchParams.get(Licenses.MAIL_PARAM)).toBe("failed");
    });

    it("should show a restore that saved without its email", () => {
      renderLicense({ mailFailed: true });
      expect(
        screen.getByText('"Test rig" is restored, but the email about it failed.'),
      ).toBeTruthy();
    });

    it("should show a revoke that saved without its email", () => {
      renderLicense({ license: REVOKED, mailFailed: true });
      expect(
        screen.getByText('"Test rig" is revoked, but the email about it failed.'),
      ).toBeTruthy();
    });

    it("should drop the mark from the URL so a later reload hides the message", () => {
      window.history.replaceState(null, "", "/licenses/abc?org=acme&mail=failed");
      renderLicense({ mailFailed: true });
      expect(window.location.search).toBe("?org=acme");
    });
  });
});
