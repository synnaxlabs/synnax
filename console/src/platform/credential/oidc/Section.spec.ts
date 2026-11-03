// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type auth,
  connection,
  type ontology,
  rack,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
  createTestUser,
} from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { clearMocks, mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  onTestFinished,
  vi,
} from "vitest";

const runtime = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, runtime);
});

import {
  LOCAL_SUBJECT,
  type LocalInvite,
  type OIDCServer,
  pointAtOIDC,
  serveOIDC,
  type ServeOIDCParams,
} from "@/platform/core/testutil";
import { Credential } from "@/platform/credential";
import { findButton, openModal } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import {
  CaptureStatuses,
  type CreateConsoleWrapperParams,
  createStatusConsoleWrapper,
  disableActEnvironment,
  getIconButton,
  getIconButtons,
} from "@/testutil";

// the local auth server answers outside any act
disableActEnvironment();

const PROVIDER: auth.OIDCLogin = {
  provider: "company-sso",
  authorizationEndpoint: "https://sso.example.com/authorize",
  clientId: "synnax",
};

const OTHER_PROVIDER: auth.OIDCLogin = { ...PROVIDER, provider: "other-sso" };

const createLink = (subject: ontology.ID): auth.oidc.Link => ({
  key: crypto.randomUUID(),
  subject,
  provider: PROVIDER.provider,
  account: "ada@example.com",
});

const CORE = createCore("local");

const createInvite = (subject: ontology.ID): LocalInvite => ({
  key: crypto.randomUUID(),
  subject,
  expiry: 1,
});

const OTHER_USER = user.ontologyID("9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a");

// A client of a local server that has opened its session, so it knows its subject.
const open = async (params?: ServeOIDCParams) => {
  const server = await serveOIDC(params);
  const client = createTestClient(server.address);
  await client.auth.oidc.retrieve();
  return { client, server };
};

// the accepted methods hide the section until the methods answer arrives
const serveAnswered = async (params: ServeOIDCParams) => {
  let served!: () => void;
  const answered = new Promise<void>((resolve) => (served = resolve));
  const { client } = await open({ ...params, onServed: () => served() });
  return { client, answered };
};

// a local server cannot hold the streams a connected client opens
const createWrapper = async (params: CreateConsoleWrapperParams) =>
  await createStatusConsoleWrapper({
    ...params,
    status: connection.DEFAULT_STATUS,
    preloadedState: {
      [Session.Core.SLICE_NAME]: {
        ...Session.Core.ZERO_SLICE_STATE,
        cores: { [CORE.key]: CORE },
        selected: CORE.key,
      },
    },
  });

const openLocal = async (client: Synnax, subject: ontology.ID, extra?: ReactNode) =>
  await openModal(Credential.useModal, {
    client,
    params: { subject },
    extra,
    createWrapper,
  });

const expectNoSection = async (): Promise<void> =>
  await expectAlways(() => expect(screen.queryByText("Provider accounts")).toBeNull());

describe("OIDC section", () => {
  // jsdom serves no secure context; a provider link needs one
  const setSecureContext = (secure: boolean) =>
    Object.defineProperty(window, "isSecureContext", {
      value: secure,
      configurable: true,
    });

  beforeEach(() => setSecureContext(true));

  afterEach(() => {
    setSecureContext(false);
    runtime.engine = "web";
  });

  it("should hide when the Core does not accept OIDC", async () => {
    const { client, answered } = await serveAnswered({ methods: ["password"] });
    await openLocal(client, LOCAL_SUBJECT);
    await answered;
    await expectNoSection();
  });

  it("should hide for a rack", async () => {
    const { client, answered } = await serveAnswered({});
    await openLocal(client, rack.ontologyID(1));
    await answered;
    await expectNoSection();
  });

  it("should list the links and offer the providers left to link", async () => {
    const { client } = await open({
      providers: [PROVIDER, OTHER_PROVIDER],
      links: [createLink(LOCAL_SUBJECT)],
    });
    await openLocal(client, LOCAL_SUBJECT);
    await screen.findByText("ada@example.com");
    expect(screen.getByText(PROVIDER.provider)).toBeTruthy();
    await screen.findByText(`Link ${OTHER_PROVIDER.provider}`);
    expect(screen.queryByText(`Link ${PROVIDER.provider}`)).toBeNull();
  });

  it("should unlink the caller's own account", async () => {
    const { client, server } = await open({ links: [createLink(LOCAL_SUBJECT)] });
    await openLocal(client, LOCAL_SUBJECT);
    await screen.findByText("ada@example.com");
    fireEvent.click(getIconButton(document.body, "delete"));
    await screen.findByText("No linked accounts");
    expect(server.links).toEqual([]);
    expect(await screen.findByText("Successfully unlinked account")).toBeTruthy();
  });

  it("should mark a browser Link as a link for the Core it started on", async () => {
    const { client } = await open({ providers: [PROVIDER] });
    await openLocal(client, LOCAL_SUBJECT);
    fireEvent.click(await screen.findByText(`Link ${PROVIDER.provider}`));
    onTestFinished(() => sessionStorage.clear());
    await waitFor(() =>
      expect(
        JSON.parse(sessionStorage.getItem("synnax-oidc-login") ?? "{}"),
      ).toMatchObject({ purpose: "link", core: CORE.key }),
    );
  });

  it("should offer no Link on another user", async () => {
    const { client } = await open({
      providers: [PROVIDER, OTHER_PROVIDER],
      links: [createLink(OTHER_USER)],
    });
    await openLocal(client, OTHER_USER);
    await screen.findByText("ada@example.com");
    await expectAlways(() =>
      expect(screen.queryByText(`Link ${OTHER_PROVIDER.provider}`)).toBeNull(),
    );
  });

  describe("invites", () => {
    const CREATE = "Create invite";
    const REPLACE = "Replace invite";

    it("should hide when the Core does not accept OIDC", async () => {
      const { client, answered } = await serveAnswered({
        methods: ["password"],
        userActions: ["update"],
      });
      await openLocal(client, OTHER_USER);
      await answered;
      await expectAlways(() => expect(screen.queryByText(CREATE)).toBeNull());
    });

    it.each([
      ["another user", OTHER_USER],
      ["the caller", LOCAL_SUBJECT],
    ])("should hide without update permission on %s", async (_, subject) => {
      const { client, server } = await open({ userActions: ["retrieve"] });
      await openLocal(client, subject);
      await screen.findByText("Provider accounts");
      await waitFor(() => expect(server.policyReads).toBeGreaterThan(0));
      await expectAlways(() => expect(screen.queryByText(CREATE)).toBeNull());
    });

    it("should show a new invite's code until the dialog closes", async () => {
      const { client, server } = await open({ userActions: ["update"] });
      await openLocal(client, OTHER_USER);
      fireEvent.click(await screen.findByText(CREATE));
      await waitFor(() => expect(server.invites).toHaveLength(1));
      const token = `syi_${server.invites[0].key}`;
      expect(await screen.findByDisplayValue(token)).toBeTruthy();
      expect(
        screen.getByText("Copy this invite code now. You won't see it again."),
      ).toBeTruthy();
      expect(await screen.findByText("Successfully created invite")).toBeTruthy();
      cleanup();
      await openLocal(client, OTHER_USER);
      await screen.findByText(REPLACE);
      expect(screen.queryByDisplayValue(token)).toBeNull();
    });

    it("should revoke a live invite and hide its code", async () => {
      const old = createInvite(OTHER_USER);
      const { client, server } = await open({
        userActions: ["update"],
        invites: [old],
      });
      await openLocal(client, OTHER_USER);
      expect(await screen.findByText(/^Invite expires/)).toBeTruthy();
      fireEvent.click(await screen.findByText(REPLACE));
      await waitFor(() => expect(server.invites[0].key).not.toBe(old.key));
      const token = `syi_${server.invites[0].key}`;
      await screen.findByDisplayValue(token);
      fireEvent.click(await screen.findByText("Revoke"));
      await screen.findByText(CREATE);
      expect(server.invites).toEqual([]);
      expect(screen.queryByText(/^Invite expires/)).toBeNull();
      expect(screen.queryByDisplayValue(token)).toBeNull();
      expect(await screen.findByText("Successfully revoked invite")).toBeTruthy();
    });

    it("should say that a new invite replaces a live one", async () => {
      const old = createInvite(OTHER_USER);
      const { client, server } = await open({
        userActions: ["update"],
        invites: [old],
      });
      await openLocal(client, OTHER_USER);
      fireEvent.click(await screen.findByText(REPLACE));
      expect(screen.queryByText(CREATE)).toBeNull();
      await waitFor(() => expect(server.invites[0].key).not.toBe(old.key));
      expect(server.invites).toHaveLength(1);
    });
  });

  describe("with a Core's permissions", () => {
    const openOther = async (client: Synnax) => {
      const { user: other } = await createTestUser(createTestClient());
      const subject = user.ontologyID(other.key);
      const server = await serveOIDC({ links: [createLink(subject)] });
      pointAtOIDC(client, server);
      await openModal(Credential.useModal, { client, params: { subject } });
      await screen.findByText("ada@example.com");
      return server;
    };

    it("should unlink another user's account with delete permission", async () => {
      const server = await openOther(createTestClient());
      await waitFor(() =>
        expect(getIconButtons(document.body, "delete")).toHaveLength(1),
      );
      fireEvent.click(getIconButton(document.body, "delete"));
      await screen.findByText("No linked accounts");
      expect(server.links).toEqual([]);
    });

    it("should offer no unlink without delete permission", async () => {
      const viewer = await createTestClientWithRole(createTestClient(), "Viewer");
      await openOther(viewer);
      await expectAlways(() =>
        expect(getIconButtons(document.body, "delete")).toHaveLength(0),
      );
    });
  });

  describe("on desktop", () => {
    beforeEach(() => {
      runtime.engine = "tauri";
      mockWindows("main");
      mockIPC((cmd, args) => {
        if (cmd === "oidc_listen") return 4242;
        if (cmd === "oidc_wait")
          return `code=abc&state=${(args as { state: string }).state}`;
        return undefined;
      });
      onTestFinished(clearMocks);
    });

    const clickLink = async (server: OIDCServer, client: Synnax) => {
      let statuses: Status.NotificationSpec[] = [];
      await openLocal(
        client,
        LOCAL_SUBJECT,
        createElement(CaptureStatuses, { onStatuses: (s) => (statuses = s) }),
      );
      await screen.findByText(`Link ${PROVIDER.provider}`);
      fireEvent.click(findButton(`Link ${PROVIDER.provider}`));
      await waitFor(() => expect(server.linked).toHaveLength(1));
      return () => statuses.map((s) => s.message);
    };

    it("should link the account over the loopback redirect", async () => {
      const { client, server } = await open({ providers: [PROVIDER] });
      await clickLink(server, client);
      await screen.findByText("new-account");
      expect(server.linked[0]).toMatchObject({
        provider: PROVIDER.provider,
        code: "abc",
        redirect_uri: "http://127.0.0.1:4242/callback",
      });
      expect(await screen.findByText("Successfully linked account")).toBeTruthy();
    });

    it("should report a link the Core refuses", async () => {
      const { client, server } = await open({
        providers: [PROVIDER],
        linkRefused: true,
      });
      const statuses = await clickLink(server, client);
      await waitFor(() =>
        expect(statuses()).toContain("Failed to link provider account"),
      );
    });
  });
});
