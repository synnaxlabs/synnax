// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/feature/task";
import { createTestDevice } from "@/platform/device/testutil";
import {
  checkInUse,
  renderForward,
  settleWrites,
} from "@/platform/notifications/testutil";
import { createTaskStatus } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

// The Core creates only the task types it knows.
const SCANNER: Task.Scanner = { make: "ethercat", scanType: "ethercat_scan" };
const OTHER_SCAN_TYPE = "opc_scan";

const RULE = Task.createNotificationRule([SCANNER]);

const createRack = async (): Promise<rack.Rack> =>
  await client.racks.create({ name: uniqueName("rack") });

const createTask = async (
  r: rack.Rack,
  { type = SCANNER.scanType, internal = true }: { type?: string; internal?: boolean },
): Promise<task.Task> =>
  await r.createTask({ name: uniqueName("task"), type, config: {}, internal });

describe("Task.createNotificationRule", () => {
  describe("kind", () => {
    it("should call a status that answers a command a reply", () => {
      expect(RULE.kind(createTaskStatus({ details: { cmd: "start-1" } }))).toBe(
        "reply",
      );
    });

    it("should call a status that answers no command a state", () => {
      expect(RULE.kind(createTaskStatus())).toBe("state");
    });
  });

  describe("inUse", () => {
    it("should hold a task that the user made in use", async () => {
      const tsk = await createTask(await createRack(), { internal: false });
      expect(await checkInUse(client, RULE, tsk.key)).toBe(true);
    });

    it("should hold a scanner in use with a configured device it scans", async () => {
      const r = await createRack();
      const scanner = await createTask(r, {});
      await createTestDevice(client, { make: SCANNER.make, rack: r.key });
      expect(await checkInUse(client, RULE, scanner.key)).toBe(true);
    });

    it("should not hold a scanner without a configured device in use", async () => {
      const r = await createRack();
      const scanner = await createTask(r, {});
      await createTestDevice(client, {
        make: SCANNER.make,
        rack: r.key,
        configured: false,
      });
      await createTestDevice(client, { make: uniqueName("make"), rack: r.key });
      await createTestDevice(client, {
        make: SCANNER.make,
        rack: (await createRack()).key,
      });
      expect(await checkInUse(client, RULE, scanner.key)).toBe(false);
    });

    it("should hold an internal task of an unknown scan type in use", async () => {
      const tsk = await createTask(await createRack(), { type: OTHER_SCAN_TYPE });
      expect(await checkInUse(client, RULE, tsk.key)).toBe(true);
    });

    it("should not hold a task that no longer exists in use", async () => {
      expect(await checkInUse(client, RULE, uuid.create())).toBe(false);
    });
  });

  describe("onUseChange", () => {
    it("should list a scanner problem once it scans a configured device", async () => {
      const r = await createRack();
      const { result } = await renderForward(client, { task: RULE });
      const scanner = await createTask(r, {});
      const dev = await createTestDevice(client, {
        make: SCANNER.make,
        rack: r.key,
        configured: false,
      });
      const { key, message } = await client.statuses.set(
        createTaskStatus({
          key: task.statusKey(scanner.key),
          variant: "error",
          message: uniqueName("message"),
          details: { task: scanner.key },
        }),
      );
      await settleWrites(client);
      await client.devices.create({ ...dev, configured: true });
      await waitFor(() =>
        expect(result.current.map(({ status: s }) => s.key)).toContain(key),
      );
      expect(screen.queryByText(message)).toBeNull();
    });
  });

  describe("coveredBy", () => {
    const coveredBy = (rackKey: rack.Key): string | null => {
      if (RULE.coveredBy == null) throw new Error("the task rule must name a cover");
      return RULE.coveredBy(createTaskStatus({ details: { rack: rackKey } }));
    };

    it("should name the status of the task's rack", () => {
      expect(coveredBy(7)).toBe(rack.statusKey(7));
    });

    it("should name nothing for a status with no rack", () => {
      expect(coveredBy(0)).toBeNull();
    });
  });
});
