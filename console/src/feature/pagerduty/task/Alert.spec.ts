// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PagerDuty } from "@/feature/pagerduty";
import { type Task } from "@/platform/task";
import {
  awaitTaskKey,
  clickConfigure,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const renderAlert = async (
  options: { client?: Synnax | null; params?: Task.FormViewParams } = {},
) => await renderTaskFormTab(PagerDuty.Task.Alert, PagerDuty.Task.ALERT_TYPE, options);

const ROUTING_KEY_PLACEHOLDER = "R022XIJR9M266DX570EVE6EXP1AFBN6D";

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

  it("should seed the form from a valid config passed through view args", async () => {
    const config = createAlertConfig();
    await renderAlert({ params: { config } });
    await screen.findByDisplayValue("R".repeat(32));
    await screen.findByText("New alert");
  });

  it("should fall back to the zero config when the view args config is invalid", async () => {
    const config = createAlertConfig({ routingKey: "too_short" });
    await renderAlert({ params: { config } });
    const input = await screen.findByPlaceholderText<HTMLInputElement>(
      ROUTING_KEY_PLACEHOLDER,
    );
    expect(input.value).toBe("");
    expect(screen.queryByText("New alert")).toBeNull();
  });

  describe("onConfigure against a live cluster", () => {
    const client = createTestClient();

    it("should create the alert task on the rack from the view args", async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const config = createAlertConfig();
      const rendered = await renderAlert({
        client,
        params: { rackKey: rack.key, config },
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(rendered);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: PagerDuty.Task.ALERT_SCHEMAS,
      });
      expect(created.type).toBe(PagerDuty.Task.ALERT_TYPE);
      expect(created.rack).toBe(rack.key);
      expect(created.config.routingKey).toBe(config.routingKey);
      expect(created.config.alerts).toHaveLength(1);
      expect(created.config.alerts[0].status).toBe(config.alerts[0].status);
    });
  });
});
