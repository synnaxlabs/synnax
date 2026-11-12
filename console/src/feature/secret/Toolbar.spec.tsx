// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  certificate,
  NotFoundError,
  secret,
  type Synnax as Client,
  user,
} from "@synnaxlabs/client";
import { CERTIFICATE, createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { Access } from "@synnaxlabs/pluto";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, assert, describe, expect, it } from "vitest";

import { Secret } from "@/feature/secret";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  assertDefined,
  awaitGranted,
  awaitTextEditing,
  CaptureStatuses,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  createTestClientWithGrants,
  getIconButton,
  openContextMenu,
  renderHookWithConsole,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

// The toolbar lists every secret and certificate on the Core, so each test removes
// what it created.
const createdSecrets: secret.Key[] = [];
const createdCertificates: certificate.Key[] = [];

// Deletion tests already removed their secret, so a NotFound here is expected.
const ignoreNotFound = (err: unknown) => {
  if (!NotFoundError.matches(err)) throw new Error("cleanup failed", { cause: err });
};

afterEach(async () => {
  // Unmount first, or the toolbar would watch the deletions below.
  cleanup();
  for (const key of createdCertificates.splice(0))
    await client.certificates.delete(key).catch(ignoreNotFound);
  for (const key of createdSecrets.splice(0))
    await client.secrets.delete(key).catch(ignoreNotFound);
});

const createSecret = async (): Promise<secret.Secret> => {
  const s = await client.secrets.set({
    name: uniqueName("secret"),
    source: "stored",
    value: "hunter2",
  });
  createdSecrets.push(s.key);
  return s;
};

/** Creates a certificate with a private key, which makes the key a used secret. */
const createCertificate = async () => {
  const c = await client.certificates.set({
    name: uniqueName("certificate"),
    source: "stored",
    ...CERTIFICATE,
  });
  assert(c.source === "stored" && c.privateKey != null);
  createdCertificates.push(c.key);
  createdSecrets.push(c.privateKey);
  return { certificate: c, privateKey: await client.secrets.retrieve(c.privateKey) };
};

const renderToolbar = async (as: Client = client) => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  let statuses: Status.NotificationSpec[] = [];
  render(
    <>
      {Secret.TOOLBAR.content}
      <Modals.Stack />
      <CaptureStatuses onStatuses={(s) => (statuses = s)} />
    </>,
    { wrapper },
  );
  return { statuses: () => statuses };
};

const retrieveByName = async (name: string): Promise<secret.Secret[]> =>
  (await client.secrets.retrieve({})).filter((s) => s.name === name);

const confirmDelete = async (name: string): Promise<void> => {
  await openContextMenu(name);
  await clickAndSettle("Delete");
  await screen.findByText(`Are you sure you want to delete ${name}?`);
  fireEvent.click(findButton("Delete"));
};

describe("secret/Toolbar", () => {
  it("creates a secret from the header action", async () => {
    await renderToolbar();
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    const name = uniqueName("created");
    fireEvent.change(await screen.findByLabelText("Secret name"), {
      target: { value: name },
    });
    const value = screen.getByPlaceholderText("Value");
    expect(value).toHaveProperty("type", "password");
    fireEvent.change(value, { target: { value: "hunter2" } });
    fireEvent.click(findButton("Create"));
    const [created] = await waitFor(async () => {
      const named = await retrieveByName(name);
      expect(named).toHaveLength(1);
      return named;
    });
    createdSecrets.push(created.key);
    expect(created.source).toBe("stored");
    await screen.findByText(name);
  });

  it("renames a secret through the inline editor", async () => {
    const s = await createSecret();
    await renderToolbar();
    await openContextMenu(s.name);
    await clickAndSettle("Rename");
    const editor = await awaitTextEditing(`${s.key}-name`);
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.secrets.retrieve(s.key)).name).toBe(renamed),
    );
  });

  it("replaces the value of a secret in place", async () => {
    const s = await createSecret();
    await renderToolbar();
    await openContextMenu(s.name);
    await clickAndSettle("Replace value");
    fireEvent.change(await screen.findByPlaceholderText("New value"), {
      target: { value: "hunter3" },
    });
    fireEvent.click(findButton("Replace"));
    await waitFor(async () => {
      const replaced = await client.secrets.retrieve(s.key);
      expect(replaced.updated.after(s.updated)).toBe(true);
    });
    expect(await retrieveByName(s.name)).toHaveLength(1);
  });

  it("keeps a rename that another user makes while the replacement is open", async () => {
    const s = await createSecret();
    await renderToolbar();
    await openContextMenu(s.name);
    await clickAndSettle("Replace value");
    const renamed = uniqueName("renamed");
    await createTestClient().secrets.rename(s.key, renamed);
    // The list and the dialog title both show the new name.
    await waitFor(() => expect(screen.getAllByText(renamed)).toHaveLength(2));
    fireEvent.change(await screen.findByPlaceholderText("New value"), {
      target: { value: "hunter3" },
    });
    fireEvent.click(findButton("Replace"));
    await waitFor(async () => {
      const replaced = await client.secrets.retrieve(s.key);
      expect(replaced.updated.after(s.updated)).toBe(true);
      expect(replaced.name).toBe(renamed);
    });
  });

  it("offers a replacement to a user who can update but not create secrets", async () => {
    const s = await createSecret();
    const editor = await createTestClientWithGrants(client, {
      retrieve: [secret.TYPE_RESOURCE_ID],
      update: [secret.TYPE_RESOURCE_ID],
    });
    await renderToolbar(editor);
    await openContextMenu(s.name);
    await screen.findByText("Replace value");
  });

  it("offers no replacement for a file secret", async () => {
    const s = await client.secrets.set({
      name: uniqueName("secret"),
      source: "file",
      fileName: "token",
    });
    createdSecrets.push(s.key);
    await renderToolbar();
    await openContextMenu(s.name);
    await screen.findByText("Rename");
    expect(screen.queryByText("Replace value")).toBeNull();
  });

  it("deletes a secret after confirmation", async () => {
    const s = await createSecret();
    await renderToolbar();
    await confirmDelete(s.name);
    await waitFor(async () => {
      await expect(client.secrets.retrieve(s.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("reports an error and keeps a secret that a certificate uses", async () => {
    const { certificate: c, privateKey } = await createCertificate();
    const { statuses } = await renderToolbar();
    await confirmDelete(privateKey.name);
    const failed = await waitFor(() => {
      const status = statuses().find((s) => s.variant === "error");
      if (status == null) throw new Error("no error status was raised");
      return status;
    });
    expect(failed.message).toBe("Failed to delete secret");
    expect(failed.description).toBe(
      `cannot delete secret ${privateKey.name} because certificate ${c.name} ` +
        "uses it as its private_key: validation error",
    );
    expect(await client.secrets.retrieve(privateKey.key)).toEqual(privateKey);
  });

  it("renames a certificate through the inline editor", async () => {
    const { certificate: c } = await createCertificate();
    await renderToolbar();
    await openContextMenu(c.name);
    await clickAndSettle("Rename");
    const editor = await awaitTextEditing(`${c.key}-name`);
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.certificates.retrieve(c.key)).name).toBe(renamed),
    );
  });

  it("deletes a certificate after confirmation", async () => {
    const { certificate: c } = await createCertificate();
    await renderToolbar();
    await confirmDelete(c.name);
    await waitFor(async () => {
      await expect(client.certificates.retrieve(c.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("lists only certificates to a user who cannot read secrets", async () => {
    const { certificate: c, privateKey } = await createCertificate();
    const reader = await createTestClientWithGrants(client, {
      retrieve: [certificate.TYPE_RESOURCE_ID],
    });
    await renderToolbar(reader);
    await screen.findByText(c.name);
    expect(screen.queryByText("Secrets")).toBeNull();
    expect(screen.queryByText(privateKey.name)).toBeNull();
  });

  it("shows the toolbar to a user who can read only certificates", async () => {
    assertDefined(Secret.TOOLBAR.useVisible);
    const { result } = await renderHookWithConsole(Secret.TOOLBAR.useVisible, {
      client: await createTestClientWithGrants(client, {
        retrieve: [certificate.TYPE_RESOURCE_ID],
      }),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });

  it("hides the toolbar from a user who can read no credentials", async () => {
    const { useVisible } = Secret.TOOLBAR;
    assertDefined(useVisible);
    const { result } = await renderHookWithConsole(
      () => ({
        visible: useVisible(),
        resolved: Access.useRetrieveGranted(user.TYPE_RESOURCE_ID),
      }),
      { client: await createTestClientWithGrants(client) },
    );
    await waitFor(() => expect(result.current.resolved).toBe(true));
    expect(result.current.visible).toBe(false);
  });

  it("offers to create a secret when there are no credentials", async () => {
    const creator = await createTestClientWithGrants(client, {
      create: [secret.TYPE_RESOURCE_ID],
    });
    await renderToolbar(creator);
    await screen.findByText("No credentials");
    fireEvent.click(await screen.findByText("Create secret"));
    await screen.findByLabelText("Secret name");
  });

  it("offers no create action to a user who cannot create secrets", async () => {
    const outsider = await createTestClientWithGrants(client);
    await awaitGranted(outsider, user.TYPE_RESOURCE_ID, "retrieve");
    await renderToolbar(outsider);
    await screen.findByText("No credentials");
    expect(screen.queryByText("Create secret")).toBeNull();
  });
});
