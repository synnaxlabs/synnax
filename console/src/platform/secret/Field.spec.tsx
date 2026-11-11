// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { secret, type Synnax as Client } from "@synnaxlabs/client";
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

import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
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
  onChange: (key: string) => void;
  mode?: Form.Mode;
}

const Fixture = ({
  value,
  defaultName,
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
      <Secret.Field path={path.secret} defaultName={defaultName} />
    </Form.Form>
  );
};
Fixture.displayName = "Fixture";

interface RenderedField {
  defaultName: string;
  onChange: Mock<(key: string) => void>;
}

const renderField = async (
  value: string = "",
  as: Client = client,
  mode?: Form.Mode,
): Promise<RenderedField> => {
  const defaultName = uniqueName("default");
  const onChange = vi.fn<(key: string) => void>();
  const { wrapper } = await createConsoleWrapper({ client: as });
  render(
    <>
      <Fixture
        value={value}
        defaultName={defaultName}
        onChange={onChange}
        mode={mode}
      />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { defaultName, onChange };
};

const open = async (shown: string): Promise<void> => {
  fireEvent.click(await findDialogTriggerByText(shown));
};

// The list pages, so a search finds a secret however many the Core holds.
const findOption = async (s: secret.Secret): Promise<HTMLElement> => {
  fireEvent.change(await screen.findByPlaceholderText("Search secrets..."), {
    target: { value: s.name },
  });
  return await screen.findByRole("option", { name: new RegExp(s.name) });
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
    const option = await findOption(s);
    within(option).getByText(Validity.describeSecret(s));
  });

  it("shows no prompt to pick a secret in preview", async () => {
    await renderField("", client, "preview");
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
    await renderField(s.key, await createTestClientWithGrants(client));
    await findDialogTriggerByText("Unavailable secret");
    expect(screen.queryByText(s.name)).toBeNull();
  });

  it("sets the value to the key of the picked secret", async () => {
    const s = await createSecret(client, created);
    const { onChange } = await renderField();
    await open("Select a secret");
    fireEvent.click(await findOption(s));
    await waitFor(() => expect(onChange).toHaveBeenLastCalledWith(s.key));
    const trigger = await findDialogTriggerByText(s.name);
    within(trigger).getByText(Validity.describeSecret(s));
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

  it("offers no new secret to a user who cannot create secrets", async () => {
    const s = await createSecret(client, created);
    const reader = await createTestClientWithGrants(client, {
      retrieve: [secret.TYPE_RESOURCE_ID],
    });
    await awaitGranted(reader, secret.TYPE_RESOURCE_ID, "retrieve");
    await renderField("", reader);
    await open("Select a secret");
    await findOption(s);
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
});
