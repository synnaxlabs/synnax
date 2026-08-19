// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import {
  fakePickedFile,
  interceptFilePicker,
  renderHookWithConsole,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

const useHarness = () => ({
  run: Import.use(),
  statuses: Status.useNotifications().statuses,
});

describe("use", () => {
  it("reads each picked file and streams its contents to the Core", async () => {
    const picker = interceptFilePicker();
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const original = await client.logs.create(proj.key, { name: uniqueName("log") });
    const stream = await client.imex.export(log.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = await new Response(stream).text();
    const { result } = await renderHookWithConsole(useHarness, {
      client,
      preloadedState: { project: { version: 0, selected: proj.key } },
    });
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("widget.json", data)]);
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({
        ids: [project.ontologyID(proj.key)],
      });
      const logs = children.filter(
        (child) => child.id.type === log.TYPE_ONTOLOGY_ID.type,
      );
      expect(logs).toHaveLength(2);
      expect(logs.map(({ name }) => name)).toEqual([original.name, original.name]);
    });
  });

  it("does nothing when the file picker is cancelled", async () => {
    const picker = interceptFilePicker();
    const { result } = await renderHookWithConsole(useHarness, {
      client: createTestClient(),
      preloadedState: { project: { version: 0, selected: "project-1" } },
    });
    act(() => result.current.run());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.cancel();
    await act(async () => {});
    expect(result.current.statuses).toHaveLength(0);
  });
});
