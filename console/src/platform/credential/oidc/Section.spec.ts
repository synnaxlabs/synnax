// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  AccountLinkedError,
  type auth,
  connection,
  type resource,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import {
  createEmptyRole,
  createTestClient,
  createTestClientWithRole,
  createTestUser,
} from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { expectAlways } from "@synnaxlabs/x/testutil";
import { clearMocks, mockIPC, mockWindows } from "@tauri-apps/api/mocks";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createElement, type FC, type PropsWithChildren } from "react";
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

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, runtime);
});

import {
  LOCAL_SUBJECT,
  type OIDCServer,
  pointAtOIDC,
  serveOIDC,
  type ServeOIDCParams,
} from "@/platform/core/server/testutil";
import { OIDC } from "@/platform/credential/oidc";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";
import {
  CaptureStatuses,
  createConsoleWrapper,
  createStatusConsoleWrapper,
  disableActEnvironment,
  getIconButtons,
} from "@/testutil";

// the local auth server answers outside any act
disableActEnvironment();

const PROVIDER: auth.OIDCLogin = {
  provider: "company-sso",
  issuer: "https://sso.example.com",
  authorizationEndpoint: "https://sso.example.com/authorize",
  clientId: "synnax",
};

const OTHER_PROVIDER: auth.OIDCLogin = {
  ...PROVIDER,
  provider: "other-sso",
  issuer: "https://other.example.com",
};

const createLink = (subject: resource.ID): auth.oidc.Link => ({
  key: crypto.randomUUID(),
  user: subject.key,
  issuer: PROVIDER.issuer,
  account: "ada@example.com",
});

const CORE = createCore("local");

const OTHER_USER = user.resourceID("9f8e7d6c-5b4a-4392-8170-6f5e4d3c2b1a");

// A client of a local server that has opened its session, so it knows its subject.
const open = async (params?: ServeOIDCParams) => {
  const server = await serveOIDC(params);
  const client = createTestClient(server.address);
  await client.auth.oidc.retrieve();
  return { client, server };
};

// The modal around the section needs a browser, which cannot start a local server.
const renderSection = (
  wrapper: FC<PropsWithChildren>,
  subject: resource.ID,
  onStatuses?: (statuses: Status.NotificationSpec[]) => void,
) => {
  const onMessage = vi.fn<(message: string | null) => void>();
  render(
    createElement(
      wrapper,
      null,
      createElement(OIDC.Section, { subject, onMessage }),
      onStatuses != null && createElement(CaptureStatuses, { onStatuses }),
    ),
  );
  return onMessage;
};

const renderLocal = async (
  client: Synnax,
  subject: resource.ID,
  onStatuses?: (statuses: Status.NotificationSpec[]) => void,
) => {
  // a local server cannot hold the streams a connected client opens
  const { wrapper } = await createStatusConsoleWrapper({
    client,
    status: connection.DEFAULT_STATUS,
    preloadedState: {
      [Session.Core.SLICE_NAME]: {
        ...Session.Core.ZERO_SLICE_STATE,
        cores: { [CORE.key]: CORE },
        selected: CORE.key,
      },
    },
  });
  return renderSection(wrapper, subject, onStatuses);
};

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

  it("should list the links and offer the providers left to link", async () => {
    const { client } = await open({
      providers: [PROVIDER, OTHER_PROVIDER],
      links: [createLink(LOCAL_SUBJECT)],
    });
    await renderLocal(client, LOCAL_SUBJECT);
    await screen.findByText("ada@example.com");
    await screen.findByText(PROVIDER.provider);
    await screen.findByText(`Link ${OTHER_PROVIDER.provider}`);
    expect(screen.queryByText(`Link ${PROVIDER.provider}`)).toBeNull();
  });

  it("should offer to unlink the caller's own account", async () => {
    const { client } = await open({ links: [createLink(LOCAL_SUBJECT)] });
    await renderLocal(client, LOCAL_SUBJECT);
    await screen.findByText("ada@example.com");
    expect(getIconButtons(document.body, "delete")).toHaveLength(1);
  });

  it("should mark a browser Link as a link for the Core it started on", async () => {
    const { client } = await open({ providers: [PROVIDER] });
    await renderLocal(client, LOCAL_SUBJECT);
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
    await renderLocal(client, OTHER_USER);
    await screen.findByText("ada@example.com");
    await screen.findByText(PROVIDER.provider);
    await expectAlways(() =>
      expect(screen.queryByText(`Link ${OTHER_PROVIDER.provider}`)).toBeNull(),
    );
  });

  describe("with a Core's permissions", () => {
    const openOther = async (client: Synnax) => {
      const owner = createTestClient();
      const { key: role } = await createEmptyRole(owner);
      const { user: other } = await createTestUser(owner, { role });
      const subject = user.resourceID(other.key);
      const server = await serveOIDC({ links: [createLink(subject)] });
      pointAtOIDC(client, server);
      const { wrapper } = await createConsoleWrapper({ client });
      renderSection(wrapper, subject);
      await screen.findByText("ada@example.com");
      return server;
    };

    // the unlink confirm needs a browser; useModal.browser.spec runs it
    it("should offer to unlink another user's account with delete permission", async () => {
      await openOther(createTestClient());
      await waitFor(() =>
        expect(getIconButtons(document.body, "delete")).toHaveLength(1),
      );
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
    // the query the provider redirects to the loopback port with
    let redirect = "code=abc";

    beforeEach(() => {
      redirect = "code=abc";
      runtime.engine = "tauri";
      mockWindows("main");
      mockIPC((cmd, args) => {
        if (cmd === "oidc_listen") return 4242;
        if (cmd === "oidc_wait")
          return `${redirect}&state=${(args as { state: string }).state}`;
        return undefined;
      });
      onTestFinished(clearMocks);
    });

    const clickLink = async (server: OIDCServer, client: Synnax) => {
      let statuses: Status.NotificationSpec[] = [];
      const onMessage = await renderLocal(client, LOCAL_SUBJECT, (s) => (statuses = s));
      await screen.findByText(`Link ${PROVIDER.provider}`);
      fireEvent.click(findButton(`Link ${PROVIDER.provider}`));
      await waitFor(() => expect(server.linked).toHaveLength(1));
      return {
        onMessage,
        messages: () => statuses.map((s) => s.message),
        failed: () => statuses.find((s) => s.message === "Failed to link account"),
      };
    };

    it("should link the account over the loopback redirect", async () => {
      const { client, server } = await open({ providers: [PROVIDER] });
      const { onMessage } = await clickLink(server, client);
      await screen.findByText("new-account");
      expect(server.linked[0]).toMatchObject({
        provider: PROVIDER.provider,
        code: "abc",
        redirect_uri: "http://127.0.0.1:4242/callback",
      });
      expect(onMessage).toHaveBeenLastCalledWith("Successfully linked account");
    });

    it("should report a cancelled link as information", async () => {
      redirect = "error=access_denied";
      const { client, server } = await open({ providers: [PROVIDER] });
      let statuses: Status.NotificationSpec[] = [];
      await renderLocal(client, LOCAL_SUBJECT, (s) => (statuses = s));
      await screen.findByText(`Link ${PROVIDER.provider}`);
      fireEvent.click(findButton(`Link ${PROVIDER.provider}`));
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({ variant: "info", message: "Link not finished" }),
        ),
      );
      expect(statuses.map((s) => s.variant)).not.toContain("error");
      expect(server.linked).toHaveLength(0);
    });

    it("should report a link the Core refuses", async () => {
      const { client, server } = await open({
        providers: [PROVIDER],
        linkRefusal: "sy.validation",
      });
      const { messages } = await clickLink(server, client);
      await waitFor(() => expect(messages()).toContain("Failed to link account"));
    });

    it("should name the provider of an account another user holds", async () => {
      const { client, server } = await open({
        providers: [PROVIDER],
        linkRefusal: AccountLinkedError.TYPE,
      });
      const { failed } = await clickLink(server, client);
      await waitFor(() =>
        expect(failed()).toMatchObject({
          description: `This ${PROVIDER.provider} account is linked to another user.`,
        }),
      );
    });
  });
});
