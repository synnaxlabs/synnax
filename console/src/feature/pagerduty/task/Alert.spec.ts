// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { pagerduty, type secret, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { type z } from "zod";

import { PagerDuty } from "@/feature/pagerduty";
import { createSecret } from "@/platform/secret/testutil";
import {
  awaitEditableForm,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { findDialogTriggerByText, uniqueName } from "@/testutil";

const client = createTestClient();

const secrets: secret.Key[] = [];

const tasks: task.Key[] = [];

afterEach(async () => {
  // Unmount first, or the form would watch its task get deleted. The Core refuses to
  // delete a secret that a task uses.
  cleanup();
  for (const key of tasks.splice(0)) await client.tasks.delete(key);
  for (const key of secrets.splice(0)) await client.secrets.delete(key);
});

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderAlert = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(PagerDuty.Task.Alert, {
    client,
    task: ZERO_DRAFT,
    ...options,
  });
  await awaitEditableForm();
  return rendered;
};

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<PagerDuty.Task.AlertSchemas> = {
  name: "PagerDuty alert task",
  type: PagerDuty.Task.ALERT_TYPE,
  config: PagerDuty.Task.ALERT_SCHEMAS.config.parse({}),
};

const newAlertItem = (): HTMLElement =>
  screen.getByRole("option", { name: /New alert/ });

const addAlert = async (): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Add alert" }));
  await waitFor(() => expect(newAlertItem()).toBeTruthy());
};

// The Core refuses an alert that names a status it does not hold.
const createAlertConfig = async (
  overrides: Partial<z.infer<PagerDuty.Task.AlertSchemas["config"]>> = {},
): Promise<z.infer<PagerDuty.Task.AlertSchemas["config"]>> => {
  const st = await client.statuses.set({
    key: uniqueName("status"),
    name: "pressure",
    variant: "warning",
    message: "Pressure high",
  });
  return {
    ...PagerDuty.Task.ALERT_SCHEMAS.config.parse({}),
    alerts: [{ ...pagerduty.alertZ.parse({}), key: "a1", status: st.key }],
    ...overrides,
  };
};

// Creates a draft whose routing key is a new secret.
const createKeyedDraft = async (
  draft: Partial<task.New<PagerDuty.Task.AlertSchemas>> = {},
) => {
  const s = await createSecret(client, secrets);
  const t = await client.tasks.create(
    { ...ZERO_DRAFT, config: await createAlertConfig({ routingKey: s.key }), ...draft },
    PagerDuty.Task.ALERT_SCHEMAS,
  );
  tasks.push(t.key);
  return { secret: s, task: t };
};

describe("PagerDuty Alert form", () => {
  it("should show the empty state and add + select an alert", async () => {
    await renderAlert();
    await screen.findByText("No alerts");
    expect(screen.getByText("No alert selected")).toBeTruthy();
    await addAlert();
    expect(screen.getByText("Treat error as critical")).toBeTruthy();
    expect(screen.getByPlaceholderText("engine-1")).toBeTruthy();
    expect(screen.getByPlaceholderText("engines")).toBeTruthy();
    expect(screen.getByPlaceholderText("engine-failure")).toBeTruthy();
    expect(screen.queryByText("No alert selected")).toBeNull();
  });

  it("should disable and re-enable alerts through the context menu", async () => {
    await renderAlert();
    await addAlert();
    fireEvent.contextMenu(newAlertItem());
    await screen.findByText("Disable");
    expect(screen.queryByText("Enable")).toBeNull();
    fireEvent.click(screen.getByText("Disable"));
    fireEvent.contextMenu(newAlertItem());
    await screen.findByText("Enable");
    expect(screen.queryByText("Disable")).toBeNull();
    fireEvent.click(screen.getByText("Enable"));
    fireEvent.contextMenu(newAlertItem());
    await screen.findByText("Disable");
  });

  it("should offer Reload Console from the alert context menu", async () => {
    await renderAlert();
    await addAlert();
    fireEvent.contextMenu(newAlertItem());
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should remove alerts through the context menu", async () => {
    await renderAlert();
    await addAlert();
    fireEvent.contextMenu(newAlertItem());
    fireEvent.click(await screen.findByText("Remove"));
    await screen.findByText("No alerts");
    expect(screen.getByText("No alert selected")).toBeTruthy();
  });

  it("should show the routing key secret from the task's config", async () => {
    const { secret: s, task: t } = await createKeyedDraft();
    await renderAlert({ taskKey: t.key });
    await findDialogTriggerByText(s.name);
    await screen.findByRole("option", { name: /pressure/ });
  });

  it("should load a config without a routing key", async () => {
    const draft = await client.tasks.create({
      ...ZERO_DRAFT,
      config: await createAlertConfig(),
    });
    await renderAlert({ taskKey: draft.key });
    await findDialogTriggerByText("Select a secret");
    await screen.findByRole("option", { name: /pressure/ });
  });

  it("should name a new routing key secret after the task", async () => {
    const name = uniqueName("alerts");
    await renderAlert({ task: { ...ZERO_DRAFT, name } });
    fireEvent.click(await findDialogTriggerByText("Select a secret"));
    fireEvent.click(await screen.findByText("New secret"));
    const input = await screen.findByLabelText("Secret name");
    expect(input).toHaveProperty("value", `${name} routing key`);
  });

  describe("deploying against a live Core", () => {
    it("should start the alert task on the rack stored on its row", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const { secret: s, task: t } = await createKeyedDraft({ rack: rack.key });
      const { container } = await renderAlert({ taskKey: t.key });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        t.key,
        PagerDuty.Task.ALERT_SCHEMAS,
      );
      expect(deployed.type).toBe(PagerDuty.Task.ALERT_TYPE);
      expect(deployed.rack).toBe(rack.key);
      expect(deployed.config.routingKey).toBe(s.key);
      expect(deployed.config.alerts).toHaveLength(1);
      expect(deployed.config.alerts[0].status).toBe(t.config.alerts[0].status);
    });
  });
});
