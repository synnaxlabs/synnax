// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, rack, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderForward } from "@/app/notifications/testutil";
import { Status } from "@/feature/status";
import { findProblem } from "@/feature/status/testutil";
import { createTestDevice } from "@/platform/device/testutil";
import { createRackKey, writeStatus } from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Notifications.Provider", () => {
  describe("problems view", () => {
    it("should list a task problem under the problem of its rack", async () => {
      const r = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await r.createTask({
        name: uniqueName("task"),
        type: "pagerduty_alert",
        config: {},
      });
      const rackKey = await createRackKey(client);
      await renderForward(client, Status.TOOLBAR.content);
      const rackMessage = await writeStatus(client, rack.statusKey(rackKey), "error");
      const taskMessage = await writeStatus(client, task.statusKey(tsk.key), "error", {
        task: tsk.key,
        running: false,
        rack: rackKey,
      });
      const rackProblem = await findProblem(rackMessage);
      const taskProblem = await findProblem(taskMessage);
      expect(rackProblem.nextElementSibling).toBe(taskProblem);
      expect(taskProblem.classList).toContain("console--nested");
      expect(rackProblem.classList).not.toContain("console--nested");
    });

    it("should not list a reply or a device that is not configured", async () => {
      const r = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await r.createTask({
        name: uniqueName("task"),
        type: "pagerduty_alert",
        config: {},
      });
      const unused = await createTestDevice(client, { configured: false });
      const used = await createTestDevice(client);
      await renderForward(client, Status.TOOLBAR.content);
      const replyMessage = await writeStatus(client, task.statusKey(tsk.key), "error", {
        task: tsk.key,
        running: false,
        cmd: "start-1",
      });
      const unusedMessage = await writeStatus(
        client,
        device.statusKey(unused.key),
        "error",
      );
      const usedMessage = await writeStatus(
        client,
        device.statusKey(used.key),
        "error",
      );
      await findProblem(usedMessage);
      const panel = within(screen.getByRole("tabpanel"));
      expect(panel.queryByText(unusedMessage)).toBeNull();
      expect(panel.queryByText(replyMessage)).toBeNull();
    });
  });
});
