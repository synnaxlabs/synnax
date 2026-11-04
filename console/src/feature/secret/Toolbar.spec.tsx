// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, NotFoundError, type secret } from "@synnaxlabs/client";
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
import { afterEach, assert, describe, expect, it } from "vitest";

import { Secret } from "@/feature/secret";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Tree } from "@/platform/tree";
import {
  awaitTextEditing,
  CaptureStatuses,
  clickAndSettle,
  commitTextEdit,
  createConsoleWrapper,
  getIconButton,
  openContextMenu,
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
  const s = await client.secrets.create({
    name: uniqueName("secret"),
    source: "stored",
    value: "hunter2",
  });
  createdSecrets.push(s.key);
  return s;
};

/** Creates a certificate with a private key, which makes the key a used secret. */
const createCertificate = async () => {
  const c = await client.certificates.create({
    name: uniqueName("certificate"),
    source: "stored",
    ...CERTIFICATE,
  });
  assert(c.source === "stored" && c.privateKey != null);
  createdCertificates.push(c.key);
  createdSecrets.push(c.privateKey);
  return { certificate: c, privateKey: await client.secrets.retrieve(c.privateKey) };
};

const renderToolbar = async () => {
  const { wrapper } = await createConsoleWrapper({ client });
  let statuses: Status.NotificationSpec[] = [];
  // A list the client already holds renders its secrets at once, and each one
  // suspends to read its uses.
  await act(async () => {
    render(
      <Tree.Provider items={Secret.TREE_ITEMS}>
        {Secret.TOOLBAR.content}
        <Modals.Stack />
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </Tree.Provider>,
      { wrapper },
    );
  });
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

const getOption = (name: string): HTMLElement => {
  const option = screen
    .getAllByRole("option")
    .find((o) => within(o).queryByText(name) != null);
  if (option == null) throw new Error(`no option for ${name}`);
  return option;
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
      `cannot delete secret "${privateKey.name}" because certificate ` +
        `"${c.name}" uses it: validation error`,
    );
    expect(await client.secrets.retrieve(privateKey.key)).toEqual(privateKey);
  });

  it("lists the users of a secret and selects a certificate from them", async () => {
    const { certificate: c, privateKey } = await createCertificate();
    await renderToolbar();
    // Other specs can leave used secrets on the Core, so read the count of this key.
    const count = await waitFor(() =>
      within(getOption(privateKey.name)).getByText("Used by 1"),
    );
    await act(async () => {
      fireEvent.click(count);
    });
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByText(c.name));
    await waitFor(() =>
      expect(getOption(c.name).getAttribute("aria-selected")).toBe("true"),
    );
  });
});
