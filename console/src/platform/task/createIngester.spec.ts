// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { createConsoleWrapper } from "@/testutil";

const client: Synnax = createTestClient();

const TYPE = "myTask";

const schemas = {
  type: z.literal(TYPE),
  config: z.object({ device: z.string(), sampleRate: z.number() }),
  statusData: z.unknown(),
};

const getInitialValues: Task.GetInitialValues<typeof schemas> = ({ config }) => ({
  name: "Imported Task",
  type: TYPE,
  config: schemas.config.parse(config),
});

const awaitCreateGranted = async (): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  const { result } = renderHook(() => Access.useCreateGranted(task.TYPE_ONTOLOGY_ID), {
    wrapper,
  });
  await waitFor(() => expect(result.current).toBe(true));
};

describe("createIngester", () => {
  it("should reject an invalid config without opening a tab", async () => {
    await awaitCreateGranted();
    const ingest = Task.createIngester({ getInitialValues });
    const openTab = vi.fn();
    await expect(
      ingest(
        { device: "dev-1" },
        { openTab, client, projectKey: "", fileName: "test.json" },
      ),
    ).rejects.toThrow();
    expect(openTab).not.toHaveBeenCalled();
  });

  it("should create a draft task and open its resource tab", async () => {
    await awaitCreateGranted();
    const ingest = Task.createIngester({ getInitialValues });
    const openTab = vi.fn();
    const data = { device: "dev-1", sampleRate: 100 };
    await ingest(data, { openTab, client, projectKey: "", fileName: "test.json" });
    expect(openTab).toHaveBeenCalledTimes(1);
    const opened = openTab.mock.calls[0][0];
    expect(opened.variant).toBe("resource");
    expect(opened.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const created = await client.tasks.retrieve({
      key: opened.resource.key,
      schemas,
    });
    expect(created.rack).toBe(0);
    expect(created.config).toEqual(data);
    expect(created.name).toBe("Imported Task");
  });
});
