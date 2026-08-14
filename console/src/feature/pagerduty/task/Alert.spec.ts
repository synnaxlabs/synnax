// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PagerDuty } from "@/feature/pagerduty";
import {
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const renderAlert = async (options: RenderTaskFormTabOptions = {}) =>
  await renderTaskFormTab(PagerDuty.Task.Alert, {
    task: ZERO_DRAFT,
    ...options,
  });

const ROUTING_KEY_PLACEHOLDER = "R022XIJR9M266DX570EVE6EXP1AFBN6D";

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = PagerDuty.Task.ZERO_ALERT_PAYLOAD;

const addAlert = async (): Promise<void> => {
  fireEvent.click(await screen.findByText("Add an alert"));
  await screen.findByText("New alert");
};

const createAlertConfig = (
  overrides: Partial<PagerDuty.Task.AlertTaskConfig> = {},
): PagerDuty.Task.AlertTaskConfig => ({
  ...PagerDuty.Task.ZERO_ALERT_TASK_CONFIG,
  routingKey: "R".repeat(32),
  alerts: [
    { ...PagerDuty.Task.ZERO_ALERT_CONFIG, key: "a1", status: uniqueName("status") },
  ],
  ...overrides,
});

describe("PagerDuty Alert form", () => {
  it("should show the empty state and add + select an alert", async () => {
    await renderAlert();
    await screen.findByText("No alerts.");
    expect(screen.getByText("No alert selected.")).toBeTruthy();
    expect(screen.getByPlaceholderText(ROUTING_KEY_PLACEHOLDER)).toBeTruthy();
    await addAlert();
    expect(screen.getByText("Treat error as critical")).toBeTruthy();
    expect(screen.getByPlaceholderText("engine-1")).toBeTruthy();
    expect(screen.getByPlaceholderText("engines")).toBeTruthy();
    expect(screen.getByPlaceholderText("engine-failure")).toBeTruthy();
    expect(screen.queryByText("No alert selected.")).toBeNull();
  });

  it("should disable and re-enable alerts through the context menu", async () => {
    await renderAlert();
    await addAlert();
    fireEvent.contextMenu(screen.getByText("New alert"));
    await screen.findByText("Disable");
    expect(screen.queryByText("Enable")).toBeNull();
    fireEvent.click(screen.getByText("Disable"));
    fireEvent.contextMenu(screen.getByText("New alert"));
    await screen.findByText("Enable");
    expect(screen.queryByText("Disable")).toBeNull();
    fireEvent.click(screen.getByText("Enable"));
    fireEvent.contextMenu(screen.getByText("New alert"));
    await screen.findByText("Disable");
  });

  it("should remove alerts through the context menu", async () => {
    await renderAlert();
    await addAlert();
    fireEvent.contextMenu(screen.getByText("New alert"));
    fireEvent.click(await screen.findByText("Remove"));
    await screen.findByText("No alerts.");
    expect(screen.getByText("No alert selected.")).toBeTruthy();
  });

  it("should seed the form from the task row's config", async () => {
    const client = createTestClient();
    const config = createAlertConfig();
    const draft = await client.tasks.create(
      { ...ZERO_DRAFT, config },
      PagerDuty.Task.ALERT_SCHEMAS,
    );
    await renderAlert({ client, taskKey: draft.key });
    await screen.findByDisplayValue("R".repeat(32));
    await screen.findByText("New alert");
  });

  it("should load a routing key the deploy schema would reject", async () => {
    const client = createTestClient();
    const config = createAlertConfig({ routingKey: "too_short" });
    const draft = await client.tasks.create({ ...ZERO_DRAFT, config });
    await renderAlert({ client, taskKey: draft.key });
    const input = await screen.findByPlaceholderText<HTMLInputElement>(
      ROUTING_KEY_PLACEHOLDER,
    );
    expect(input.value).toBe("too_short");
    expect(screen.queryByText("No alerts.")).toBeNull();
  });

  describe("deploying against a live cluster", () => {
    const client = createTestClient();

    it("should start the alert task on the rack stored on its row", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const config = createAlertConfig();
      const draft = await client.tasks.create(
        { ...ZERO_DRAFT, config, rack: rack.key },
        PagerDuty.Task.ALERT_SCHEMAS,
      );
      const { container } = await renderAlert({ client, taskKey: draft.key });
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        PagerDuty.Task.ALERT_SCHEMAS,
      );
      expect(created.type).toBe(PagerDuty.Task.ALERT_TYPE);
      expect(created.rack).toBe(rack.key);
      expect(created.config.routingKey).toBe(config.routingKey);
      expect(created.config.alerts).toHaveLength(1);
      expect(created.config.alerts[0].status).toBe(config.alerts[0].status);
    });
  });
});
