// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type certificate,
  NotFoundError,
  pagerduty,
  secret,
  type Synnax as Client,
  task,
} from "@synnaxlabs/client";
import { CERTIFICATE, createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, assert, describe, expect, it, vi } from "vitest";

import { PagerDuty } from "@/feature/pagerduty";
import { Secret } from "@/feature/secret";
import { Task } from "@/feature/task";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { createSecret } from "@/platform/secret/testutil";
import { Tree } from "@/platform/tree";
import {
  awaitGranted,
  awaitTextEditing,
  CaptureStatuses,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  createTestClientWithGrants,
  openContextMenu,
  resolveFocusedTab,
  selectTestProject,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

// The pages list every secret and certificate on the Core, so each test removes
// what it created.
const createdSecrets: secret.Key[] = [];
const createdCertificates: certificate.Key[] = [];
const createdTasks: task.Key[] = [];

// Deletion tests already removed their secret, so a NotFound here is expected.
const ignoreNotFound = (err: unknown) => {
  if (!NotFoundError.matches(err)) throw new Error("cleanup failed", { cause: err });
};

afterEach(async () => {
  // Unmount first, or the pages would watch the deletions below.
  cleanup();
  for (const key of createdTasks.splice(0)) await client.tasks.delete(key);
  for (const key of createdCertificates.splice(0))
    await client.certificates.delete(key).catch(ignoreNotFound);
  for (const key of createdSecrets.splice(0))
    await client.secrets.delete(key).catch(ignoreNotFound);
});

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

const renderPage = async (
  page: ReactElement,
  as: Client = client,
  items: Tree.Items = Secret.TREE_ITEMS,
) => {
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  let statuses: Status.NotificationSpec[] = [];
  render(
    <Tree.Provider items={items}>
      {page}
      <Modals.Stack />
      <CaptureStatuses onStatuses={(s) => (statuses = s)} />
    </Tree.Provider>,
    { wrapper },
  );
  return { statuses: () => statuses, store };
};

const renderSecrets = (
  as?: Client,
  items?: Tree.Items,
  onSelectCertificate = vi.fn(),
) =>
  renderPage(<Secret.Secrets onSelectCertificate={onSelectCertificate} />, as, items);

const renderCertificates = (as?: Client) => renderPage(<Secret.Certificates />, as);

const retrieveByName = async (name: string): Promise<secret.Secret[]> =>
  (await client.secrets.retrieve({})).filter((s) => s.name === name);

const confirmDelete = async (name: string): Promise<void> => {
  await openContextMenu(name);
  await clickAndSettle("Delete");
  await screen.findByText(`Are you sure you want to delete ${name}?`);
  fireEvent.click(findButton("Delete"));
};

const getOption = (name: string): HTMLElement => screen.getByRole("option", { name });

describe("Secret.Secrets", () => {
  it("creates a secret from the create button", async () => {
    await renderSecrets();
    fireEvent.click(await screen.findByText("Create secret"));
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
    const s = await createSecret(client, createdSecrets);
    await renderSecrets();
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
    const s = await createSecret(client, createdSecrets);
    await renderSecrets();
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
    const s = await createSecret(client, createdSecrets);
    await renderSecrets();
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
    const s = await createSecret(client, createdSecrets);
    const editor = await createTestClientWithGrants(client, {
      retrieve: [secret.TYPE_RESOURCE_ID],
      update: [secret.TYPE_RESOURCE_ID],
    });
    await renderSecrets(editor);
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
    await renderSecrets();
    await openContextMenu(s.name);
    await screen.findByText("Rename");
    expect(screen.queryByText("Replace value")).toBeNull();
  });

  it("deletes a secret after confirmation", async () => {
    const s = await createSecret(client, createdSecrets);
    await renderSecrets();
    await confirmDelete(s.name);
    await waitFor(async () => {
      await expect(client.secrets.retrieve(s.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("reports an error and keeps a secret that a certificate uses", async () => {
    const { certificate: c, privateKey } = await createCertificate();
    const { statuses } = await renderSecrets();
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

  it("lists the users of a secret and shows a certificate from them", async () => {
    const { certificate: c, privateKey } = await createCertificate();
    const onSelectCertificate = vi.fn();
    await renderSecrets(client, Secret.TREE_ITEMS, onSelectCertificate);
    // Other specs can leave used secrets on the Core, so read the count of this key.
    const count = await waitFor(() =>
      within(getOption(privateKey.name)).getByText("Used by 1"),
    );
    await act(async () => {
      fireEvent.click(count);
    });
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByText(c.name));
    expect(onSelectCertificate).toHaveBeenCalledWith(c.key);
  });

  it("counts the users of a secret that arrives after the list loads", async () => {
    const s = await createSecret(client, createdSecrets);
    await renderSecrets();
    await screen.findByText(s.name);
    const { privateKey } = await createCertificate();
    await waitFor(() => within(getOption(privateKey.name)).getByText("Used by 1"));
  });

  it("opens the users of a secret from the context menu", async () => {
    const s = await createSecret(client, createdSecrets);
    await renderSecrets();
    await openContextMenu(s.name);
    await act(async () => {
      fireEvent.click(await screen.findByRole("menuitem", { name: "Used by" }));
    });
    const dialog = await screen.findByRole("dialog");
    await within(dialog).findByText("Nothing uses this secret.");
  });

  it("opens the users on hover and closes them after the pointer leaves", async () => {
    const { certificate: c, privateKey } = await createCertificate();
    await renderSecrets();
    const count = await waitFor(() =>
      within(getOption(privateKey.name)).getByText("Used by 1"),
    );
    await act(async () => {
      fireEvent.pointerEnter(count);
    });
    const dialog = await screen.findByRole("dialog");
    await within(dialog).findByText(c.name);
    fireEvent.pointerLeave(count);
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  });

  it("opens a task that uses a secret from its users", async () => {
    const s = await createSecret(client, createdSecrets);
    // The Core refuses an alert that names a status it does not hold.
    const st = await client.statuses.set({
      key: uniqueName("status"),
      name: "st",
      variant: "info",
      message: "st",
    });
    const t = await client.tasks.create(
      {
        name: uniqueName("alerts"),
        type: PagerDuty.Task.ALERT_TYPE,
        config: {
          ...PagerDuty.Task.ALERT_SCHEMAS.config.parse({}),
          routingKey: s.key,
          alerts: [{ ...pagerduty.alertZ.parse({}), key: "a1", status: st.key }],
        },
      },
      PagerDuty.Task.ALERT_SCHEMAS,
    );
    createdTasks.push(t.key);
    const { store } = await renderSecrets(client, {
      ...Secret.TREE_ITEMS,
      ...Task.TREE_ITEMS,
    });
    // A task opens in a tab of the selected project.
    await selectTestProject(store, client);
    const count = await waitFor(() => within(getOption(s.name)).getByText("Used by 1"));
    await act(async () => {
      fireEvent.click(count);
    });
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByText(t.name));
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource).toEqual(task.resourceID(t.key));
  });

  it("deletes every selected secret after confirmation", async () => {
    const a = await createSecret(client, createdSecrets);
    const b = await createSecret(client, createdSecrets);
    await renderSecrets();
    for (const { name } of [a, b]) {
      const option = await waitFor(() => getOption(name));
      fireEvent.click(within(option).getByRole("checkbox", { name: "Select" }));
    }
    await openContextMenu(b.name);
    expect(screen.queryByText("Rename")).toBeNull();
    await clickAndSettle("Delete");
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => {
      for (const { key } of [a, b])
        await expect(client.secrets.retrieve(key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
    });
  });

  it("offers no create button to a user who cannot create secrets", async () => {
    const s = await createSecret(client, createdSecrets);
    const reader = await createTestClientWithGrants(client, {
      retrieve: [secret.TYPE_RESOURCE_ID],
    });
    await awaitGranted(reader, secret.TYPE_RESOURCE_ID, "retrieve");
    await renderSecrets(reader);
    await screen.findByText(s.name);
    expect(screen.queryByText("Create secret")).toBeNull();
  });
});

describe("Secret.Certificates", () => {
  it("renames a certificate through the inline editor", async () => {
    const { certificate: c } = await createCertificate();
    await renderCertificates();
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
    await renderCertificates();
    await confirmDelete(c.name);
    await waitFor(async () => {
      await expect(client.certificates.retrieve(c.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });
});
