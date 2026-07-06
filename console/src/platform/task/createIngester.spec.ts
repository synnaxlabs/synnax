// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type Synnax, task } from "@synnaxlabs/client";
import { Access, Flux, type Pluto } from "@synnaxlabs/pluto";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { createConsoleWrapper } from "@/testutil";

const client: Synnax = createTestClient();

const configSchema = z.object({ device: z.string(), sampleRate: z.number() });
const zeroLayout: Task.Layout = { ...Task.LAYOUT, type: "myTask" };

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
  it("should throw on invalid config without placing a layout", async () => {
    const store = await getGrantedFluxStore();
    const ingest = Task.createIngester(configSchema, zeroLayout);
    const placeLayout = vi.fn();
    expect(() =>
      ingest(
        { device: "dev-1" },
        {
          layout: {},
          placeLayout,
          store,
          client,
          projectKey: "",
          fileName: "test.json",
        },
      ),
    ).toThrow();
    expect(placeLayout).not.toHaveBeenCalled();
  });

  it("should place a layout carrying the parsed config when creation is granted", async () => {
    const store = await getGrantedFluxStore();
    const ingest = Task.createIngester(configSchema, zeroLayout);
    const placeLayout = vi.fn();
    const data = { device: "dev-1", sampleRate: 100 };
    void ingest(data, {
      layout: { key: "layout-1" },
      placeLayout,
      store,
      client,
      projectKey: "",
      fileName: "test.json",
    });
    expect(placeLayout).toHaveBeenCalledTimes(1);
    const placed = placeLayout.mock.calls[0][0];
    expect(placed.type).toBe("myTask");
    expect(placed.key).toBe("layout-1");
    expect(placed.args).toEqual({ config: data });
  });
});
