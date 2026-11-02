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
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";

import { type License, type Organization } from "@/server/db/schema";
import { type Held, type Owned } from "@/server/license/list";
import { HASH_A, NOW } from "@/server/license/testutil";
import { Licenses } from "@/ui/licenses";

const ACME: Organization = {
  key: crypto.randomUUID(),
  kind: "team",
  name: "Acme",
  clerkOrgID: "org_acme",
  ownerUserID: null,
  createdAt: NOW,
};

const createChoice = (license: Partial<License> = {}, seats = 0): Owned & Held => ({
  license: {
    key: crypto.randomUUID(),
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
    ...license,
  },
  organization: ACME,
  seats,
});

const MACHINE: Licenses.Machine = {
  fingerprint: [HASH_A],
  name: "Test stand",
  seat: null,
};
const FILENAME = "Test-rig-Test-stand.lic";

const renderActivate = (props: Partial<Licenses.ActivateProps> = {}): void => {
  render(
    <Licenses.Activate
      choices={[createChoice()]}
      selected={null}
      machine={MACHINE}
      damaged={false}
      handoff={null}
      email="ada@acme.com"
      {...props}
    />,
  );
};

const stubActivate = (): MockInstance<typeof fetch> => {
  const spy = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(async () =>
      Response.json({ key: "a.b.c", activation: "act", filename: FILENAME }),
    );
  return spy;
};

const sent = (spy: MockInstance<typeof fetch>): unknown => {
  const body = spy.mock.calls[0][1]?.body;
  if (typeof body !== "string") throw new Error("the request has no JSON body");
  return JSON.parse(body);
};

describe("Licenses.Activate", () => {
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

  it("should post the machine and the handoff code", async () => {
    const fetch = stubActivate();
    const choice = createChoice();
    renderActivate({ choices: [choice], handoff: "code" });
    fireEvent.click(screen.getByText("Activate"));
    expect(await screen.findByText("Test stand is activated")).toBeTruthy();
    expect(fetch).toHaveBeenCalledWith(
      `/api/licenses/${choice.license.key}/activate`,
      expect.objectContaining({ method: "POST" }),
    );
    expect(sent(fetch)).toEqual({
      name: "Test stand",
      fingerprint: [HASH_A],
      handoff: "code",
    });
  });

  it("should let a machine try a license with every seat taken", async () => {
    const fetch = stubActivate();
    renderActivate({ choices: [createChoice({ nodes: 1 }, 1)] });
    expect(
      screen.getByText(
        "Every seat is taken. Only a machine that holds one can activate.",
      ),
    ).toBeTruthy();
    fireEvent.click(screen.getByText("Activate"));
    expect(await screen.findByText("Test stand is activated")).toBeTruthy();
    expect(fetch).toHaveBeenCalledOnce();
  });

  it("should say when the machine already holds a seat", () => {
    const choice = createChoice({ nodes: 1 }, 1);
    renderActivate({
      choices: [choice],
      machine: { ...MACHINE, seat: choice.license.key },
    });
    expect(
      screen.getByText(
        "This machine already holds a seat on this license. Activating it again " +
          "keeps the seat and issues a new key.",
      ),
    ).toBeTruthy();
  });

  it("should open on the selected license", () => {
    const other = createChoice({ label: "Other rig" }, 1);
    renderActivate({ choices: [createChoice(), other], selected: other.license.key });
    expect(screen.getByText("Other rig (Acme)")).toBeTruthy();
    expect(screen.getByText("1 of 2 seats in use")).toBeTruthy();
  });

  it("should ask for a machine name before posting", async () => {
    const fetch = stubActivate();
    renderActivate({ machine: { ...MACHINE, name: "" } });
    fireEvent.click(screen.getByText("Activate"));
    expect(await screen.findByText("Name the machine")).toBeTruthy();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("should take a fingerprint by hand when the link has none", async () => {
    const fetch = stubActivate();
    renderActivate({ machine: null });
    fireEvent.click(screen.getByText("Enter a fingerprint instead"));
    fireEvent.change(screen.getByPlaceholderText(/Paste the activation link/), {
      target: { value: HASH_A },
    });
    expect(screen.getByText("1 network adapter")).toBeTruthy();
    fireEvent.change(screen.getByPlaceholderText("Test stand, site B"), {
      target: { value: "Test stand" },
    });
    fireEvent.click(screen.getByText("Activate"));
    await waitFor(() => expect(fetch).toHaveBeenCalledOnce());
    expect(sent(fetch)).toEqual({ name: "Test stand", fingerprint: [HASH_A] });
  });

  it("should ask for the fingerprint again when the link was damaged", () => {
    renderActivate({ machine: null, damaged: true });
    expect(
      screen.getByText(
        "The link holds no fingerprint the portal can read. Open it again from the " +
          "Console, or paste the fingerprint below.",
      ),
    ).toBeTruthy();
    expect(screen.getByPlaceholderText(/Paste the activation link/)).toBeTruthy();
  });

  it("should show the commands that give the key to a Core", async () => {
    stubActivate();
    renderActivate();
    fireEvent.click(screen.getByText("Activate"));
    expect(
      await screen.findByText(`export SYNNAX_LICENSE_KEY="$(cat ${FILENAME})"`),
    ).toBeTruthy();
    expect(screen.getByText(`synnax start --license-file=${FILENAME}`)).toBeTruthy();
  });

  it("should tell a user with no license to send the link on", () => {
    renderActivate({ choices: [], handoff: "code" });
    expect(screen.getByText("No license to activate")).toBeTruthy();
    expect(
      screen.getByText(
        "Send them this link. When they activate the machine, the Console gets the " +
          "license key on its own.",
      ),
    ).toBeTruthy();
  });
});
