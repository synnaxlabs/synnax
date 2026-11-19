// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, secret, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { deep, uuid } from "@synnaxlabs/x";
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
import { afterEach, describe, expect, it, type Mock, vi } from "vitest";
import { z } from "zod";

import {
  createFileRack,
  createListing,
  serveListings,
} from "@/platform/file-browser/testutil";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { findOption } from "@/platform/picker/testutil";
import { Secret } from "@/platform/secret";
import { createSecret } from "@/platform/secret/testutil";
import { Validity } from "@/platform/secret/validity";
import {
  awaitGranted,
  createConsoleWrapper,
  createTestClientWithGrants,
  findDialogTriggerByText,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const SECRETS_DIR = "/etc/synnax/secrets";

const created: secret.Key[] = [];

afterEach(async () => {
  cleanup();
  for (const key of created.splice(0)) await client.secrets.delete(key);
});

const schema = z.object({ secret: z.string() });

const path = deep.path<z.infer<typeof schema>>();

interface FixtureProps {
  value: string;
  defaultName: string;
  rack: rack.Key;
  onChange: (key: string) => void;
  mode?: Form.Mode;
}

const Fixture = ({
  value,
  defaultName,
  rack,
  onChange,
  mode,
}: FixtureProps): ReactElement => {
  const form = Form.use({
    schema,
    mode,
    values: { secret: value },
    onChange: ({ values }) => onChange(values.secret),
  });
  return (
    <Form.Form<typeof schema> {...form}>
      <Secret.Field path={path.secret} defaultName={defaultName} rack={rack} />
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

interface RenderedField {
  defaultName: string;
  onChange: Mock<(key: string) => void>;
  /** Renders the field again with another rack, as when the device's rack changes. */
  setRack: (rack: rack.Key) => void;
}

interface RenderOptions {
  as?: Client;
  mode?: Form.Mode;
  rack?: rack.Key;
}

const renderField = async (
  value: string = "",
  { as = client, mode, rack = 0 }: RenderOptions = {},
): Promise<RenderedField> => {
  const defaultName = uniqueName("default");
  const onChange = vi.fn<(key: string) => void>();
  const { wrapper } = await createConsoleWrapper({ client: as });
  const content = (fieldRack: rack.Key): ReactElement => (
    <>
      <Fixture
        value={value}
        defaultName={defaultName}
        rack={fieldRack}
        onChange={onChange}
        mode={mode}
      />
      <Modals.Stack />
    </>
  );
  const { rerender } = render(content(rack), { wrapper });
  return { defaultName, onChange, setRack: (next) => rerender(content(next)) };
};

const open = async (shown: string): Promise<void> => {
  fireEvent.click(await findDialogTriggerByText(shown));
};

const openCreate = async (shown: string): Promise<void> => {
  await open(shown);
  fireEvent.click(await screen.findByText("New secret"));
};

describe("Secret.Field", () => {
  it("lists each secret with its name and the date its value was set", async () => {
    const s = await createSecret(client, created);
    await renderField();
    await open("Select a secret");
    const option = await findOption("secret", s.name);
    within(option).getByText(Validity.describeSecret(s));
  });

  it("shows no prompt to pick a secret in preview", async () => {
    await renderField("", { mode: "preview" });
    await screen.findByText("None");
    expect(screen.queryByText("Select a secret")).toBeNull();
  });

  it("shows that the secret is not found after it is deleted", async () => {
    const s = await createSecret(client, created);
    await renderField(s.key);
    await findDialogTriggerByText(s.name);
    await client.secrets.delete(s.key);
    await findDialogTriggerByText("Secret not found");
    expect(screen.queryByText("Select a secret")).toBeNull();
  });

  it("shows that the secret is not found for a key that names none", async () => {
    await renderField(uuid.create());
    await findDialogTriggerByText("Secret not found");
  });

  it("shows that the secret is unavailable to a user who cannot retrieve it", async () => {
    const s = await createSecret(client, created);
    await renderField(s.key, { as: await createTestClientWithGrants(client) });
    await findDialogTriggerByText("Unavailable secret");
    expect(screen.queryByText(s.name)).toBeNull();
  });

  it("sets the value to the key of the picked secret and shows only its name", async () => {
    const s = await createSecret(client, created);
    const { onChange } = await renderField();
    await open("Select a secret");
    fireEvent.click(await findOption("secret", s.name));
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith(s.key));
    const trigger = await findDialogTriggerByText(s.name);
    expect(trigger.textContent).toBe(s.name);
  });

  it("opens the create dialog with the default name and focuses the value", async () => {
    const { defaultName } = await renderField();
    await openCreate("Select a secret");
    const name = await screen.findByLabelText("Secret name");
    expect(name).toHaveProperty("value", defaultName);
    await waitFor(() =>
      expect(document.activeElement).toBe(screen.getByPlaceholderText("Value")),
    );
  });

  it("marks the value as required and requires it before it saves", async () => {
    const { onChange } = await renderField();
    await openCreate("Select a secret");
    await screen.findByLabelText("Value *");
    fireEvent.click(findButton("Create"));
    await screen.findByText("Value is required");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("opens the create dialog named after the search", async () => {
    await renderField();
    await open("Select a secret");
    const name = uniqueName("searched");
    fireEvent.change(await screen.findByPlaceholderText("Search secrets..."), {
      target: { value: name },
    });
    fireEvent.click(
      await screen.findByRole("option", { name: `Create secret "${name}"` }),
    );
    expect(await screen.findByLabelText("Secret name")).toHaveProperty("value", name);
  });

  it("hides the create option for a secret name in another case", async () => {
    const s = await createSecret(client, created);
    await renderField();
    await open("Select a secret");
    fireEvent.change(await screen.findByPlaceholderText("Search secrets..."), {
      target: { value: `${s.name}x` },
    });
    await screen.findByRole("option", { name: `Create secret "${s.name}x"` });
    await findOption("secret", s.name, s.name.toUpperCase());
    expect(screen.queryByRole("option", { name: /^Create secret/ })).toBeNull();
  });

  it("offers no new secret to a user who cannot create secrets", async () => {
    const s = await createSecret(client, created);
    const reader = await createTestClientWithGrants(client, {
      retrieve: [secret.TYPE_RESOURCE_ID],
    });
    await awaitGranted(reader, secret.TYPE_RESOURCE_ID, "retrieve");
    await renderField("", { as: reader });
    await open("Select a secret");
    await findOption("secret", s.name);
    expect(screen.queryByText("New secret")).toBeNull();
  });

  it("creates the secret on save and selects it", async () => {
    const { defaultName, onChange } = await renderField();
    await openCreate("Select a secret");
    fireEvent.change(await screen.findByPlaceholderText("Value"), {
      target: { value: "hunter2" },
    });
    fireEvent.click(findButton("Create"));
    const key = await waitFor(() => {
      const [last] = onChange.mock.lastCall ?? [];
      if (last == null) throw new Error("the field did not change");
      return last;
    });
    created.push(key);
    const s = await client.secrets.retrieve(key);
    expect(s.name).toBe(defaultName);
    expect(s.source).toBe("stored");
    await findDialogTriggerByText(defaultName);
  });

  it("holds Cancel while the secret saves", async () => {
    const { onChange } = await renderField();
    await openCreate("Select a secret");
    fireEvent.change(await screen.findByPlaceholderText("Value"), {
      target: { value: "hunter2" },
    });
    fireEvent.click(findButton("Create"));
    fireEvent.click(findButton("Cancel"));
    const key = await waitFor(() => {
      const [last] = onChange.mock.lastCall ?? [];
      if (last == null) throw new Error("the field did not change");
      return last;
    });
    created.push(key);
  });

  it("leaves the value unchanged when the create dialog is canceled", async () => {
    const s = await createSecret(client, created);
    const { onChange } = await renderField(s.key);
    await openCreate(s.name);
    fireEvent.click(findButton("Cancel"));
    await waitFor(() => expect(screen.queryByLabelText("Secret name")).toBeNull());
    await act(async () => {});
    expect(onChange).not.toHaveBeenCalled();
    await findDialogTriggerByText(s.name);
  });

  it("shows a held file secret of another rack as an error under the field", async () => {
    const [own, other] = await Promise.all([
      client.racks.create({ name: uniqueName("rack") }),
      client.racks.create({ name: uniqueName("rack") }),
    ]);
    const s = await client.secrets.set({
      name: uniqueName("file"),
      source: "file",
      fileName: "token",
      rack: own.key,
    });
    created.push(s.key);
    const { setRack } = await renderField(s.key, { rack: own.key });
    const trigger = await findDialogTriggerByText(s.name);
    expect(screen.queryByText("File on another rack")).toBeNull();
    setRack(other.key);
    const shown = await screen.findByText("File on another rack");
    expect(shown.classList).toContain("lyra--status-error");
    expect(trigger.contains(shown)).toBe(false);
  });

  it("offers no file to a field with no rack", async () => {
    await renderField();
    await openCreate("Select a secret");
    await screen.findByLabelText("Secret name");
    expect(screen.queryByText("File on rack")).toBeNull();
  });

  it("creates a file secret from the secrets folder of the rack", async () => {
    const files = await createFileRack(client);
    await serveListings(client, files, [
      createListing({ secretsDir: SECRETS_DIR }),
      createListing({
        path: SECRETS_DIR,
        secret: true,
        entries: [
          { name: "certs", path: `${SECRETS_DIR}/certs`, kind: "directory" },
          {
            name: "opcua-password",
            path: `${SECRETS_DIR}/opcua-password`,
            kind: "file",
          },
        ],
      }),
    ]);
    const { defaultName, onChange } = await renderField("", { rack: files.rack.key });
    await openCreate("Select a secret");
    fireEvent.click(await screen.findByText("File on rack"));
    await open("Select a file");
    fireEvent.click(await screen.findByRole("option", { name: "opcua-password" }));
    expect(screen.queryByRole("option", { name: "certs" })).toBeNull();
    fireEvent.click(findButton("Create"));
    const key = await waitFor(() => {
      const [last] = onChange.mock.lastCall ?? [];
      if (last == null) throw new Error("the field did not change");
      return last;
    });
    created.push(key);
    expect(await client.secrets.retrieve(key)).toEqual(
      expect.objectContaining({
        name: defaultName,
        source: "file",
        fileName: "opcua-password",
      }),
    );
  });

  it("marks the file as required and requires it for a file secret", async () => {
    const files = await createFileRack(client);
    await serveListings(client, files, [createListing({ secretsDir: SECRETS_DIR })]);
    const { onChange } = await renderField("", { rack: files.rack.key });
    await openCreate("Select a secret");
    fireEvent.click(await screen.findByText("File on rack"));
    expect((await screen.findByText("File")).textContent).toBe("File *");
    fireEvent.click(findButton("Create"));
    await screen.findByText("File is required");
    expect(onChange).not.toHaveBeenCalled();
  });
  it("lists stored secrets and the file secrets of its rack, not of another", async () => {
    const [own, other] = await Promise.all([
      client.racks.create({ name: uniqueName("rack") }),
      client.racks.create({ name: uniqueName("rack") }),
    ]);
    const prefix = uniqueName("rack secret");
    const createFile = async (r: rack.Rack, name: string): Promise<secret.Secret> => {
      const s = await client.secrets.set({
        name: `${prefix} ${name}`,
        source: "file",
        fileName: "token",
        rack: r.key,
      });
      created.push(s.key);
      return s;
    };
    const onRack = await createFile(own, "own");
    const offRack = await createFile(other, "other");
    const stored = await createSecret(client, created);
    await renderField("", { rack: own.key });
    await open("Select a secret");
    await findOption("secret", stored.name);
    fireEvent.change(screen.getByPlaceholderText("Search secrets..."), {
      target: { value: prefix },
    });
    await screen.findByRole("option", { name: new RegExp(onRack.name) });
    expect(screen.queryByRole("option", { name: new RegExp(offRack.name) })).toBeNull();
  });
});
