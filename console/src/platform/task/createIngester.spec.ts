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
import { Access, Flux, type Pluto } from "@synnaxlabs/pluto";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { createConsoleWrapper } from "@/testutil";

const client: Synnax = createTestClient();

const configSchema = z.object({ device: z.string(), sampleRate: z.number() });
const TYPE = "myTask";

const getGrantedFluxStore = async (): Promise<Pluto.FluxStore> => {
  const { wrapper } = await createConsoleWrapper({ client });
  const { result } = renderHook(
    () => ({
      store: Flux.useStore<Pluto.FluxStore>(),
      granted: Access.useCreateGranted(task.TYPE_ONTOLOGY_ID),
    }),
    { wrapper },
  );
  await waitFor(() => expect(result.current.granted).toBe(true));
  return result.current.store;
};

describe("createIngester", () => {
  it("should throw on invalid config without opening a tab", async () => {
    const store = await getGrantedFluxStore();
    const ingest = Task.createIngester(configSchema, TYPE);
    const openTab = vi.fn();
    expect(() =>
      ingest({ device: "dev-1" }, { openTab, store, client, projectKey: "" }),
    ).toThrow();
    expect(openTab).not.toHaveBeenCalled();
  });

  it("should open a tab carrying the parsed config when creation is granted", async () => {
    const store = await getGrantedFluxStore();
    const ingest = Task.createIngester(configSchema, TYPE);
    const openTab = vi.fn();
    const data = { device: "dev-1", sampleRate: 100 };
    void ingest(data, { openTab, store, client, projectKey: "" });
    expect(openTab).toHaveBeenCalledTimes(1);
    const opened = openTab.mock.calls[0][0];
    expect(opened.variant).toBe("view");
    expect(opened.type).toBe(TYPE);
    expect(opened.args).toEqual({ config: data });
  });
});
