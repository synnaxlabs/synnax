// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, log, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/pluto";
import { act, render, waitFor } from "@testing-library/react";
import { type ReactElement, useEffect } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Import } from "@/platform/import";
import {
  assertDefined,
  CaptureStatuses,
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

const createImportServerContext = (
  overrides: Partial<Import.ImportServerContext> = {},
): Import.ImportServerContext => ({
  client: null,
  parent: project.ontologyID("project-1"),
  fileName: "test.json",
  ...overrides,
});

describe("importServer", () => {
  it("fails when disconnected", async () => {
    await expect(
      Import.importServer(new TextEncoder().encode("{}"), createImportServerContext()),
    ).rejects.toThrow(DisconnectedError);
  });

  it("streams typed data to the Core and returns the created resource", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const original = await client.logs.create(proj.key, { name: uniqueName("log") });
    const stream = await client.imex.export(log.ontologyID(original.key), {
      encoding: "JSON",
    });
    const data = new Uint8Array(await new Response(stream).arrayBuffer());
    const id = await Import.importServer(
      data,
      createImportServerContext({ client, parent: project.ontologyID(proj.key) }),
    );
    assertDefined(id, "server import returned no resource");
    const created = await client.logs.retrieve({ key: id.key });
    expect(created.name).toBe(original.name);
  });

  it("routes a typeless legacy state to the Core, naming it after the file", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    // A legacy Console log state: version-stamped, no type, no name. The server
    // recognizes it by its frozen channels-array marker and names it after the file.
    const state = { version: "0.0.0", channels: [1, 2, 3], remoteCreated: false };
    const id = await Import.importServer(
      new TextEncoder().encode(JSON.stringify(state)),
      createImportServerContext({
        client,
        parent: project.ontologyID(proj.key),
        fileName: "Legacy Log.json",
      }),
    );
    assertDefined(id, "server import returned no resource");
    const created = await client.logs.retrieve({ key: id.key });
    expect(created.name).toBe("Legacy Log");
  });

  it("migrates a legacy task config through the Core on import", async () => {
    const client = createTestClient();
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    // A legacy Console task export: the camelCase config with only a type marker.
    const legacy = {
      type: "pagerduty_alert",
      routingKey: "R016395AF23B4E62B7A2BF7B24C1EF31",
      autoStart: true,
      alerts: [],
    };
    const id = await Import.importServer(
      new TextEncoder().encode(JSON.stringify(legacy)),
      createImportServerContext({
        client,
        parent: project.ontologyID(proj.key),
        fileName: "PD Alerts.json",
      }),
    );
    assertDefined(id, "server import returned no resource");
    const created = await client.tasks.retrieve({
      key: id.key,
      schemas: {
        type: z.literal("pagerduty_alert"),
        config: z.looseObject({ routingKey: z.string(), autoStart: z.boolean() }),
        statusData: z.unknown(),
      },
    });
    expect(created.name).toBe("PD Alerts");
    expect(created.rack).toBe(0);
    expect(created.config.routingKey).toBe("R016395AF23B4E62B7A2BF7B24C1EF31");
    expect(created.config.autoStart).toBe(true);
  });
});

interface HarnessProps {
  onReady: (run: (projectKey?: string) => void) => void;
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
}

const Inner = ({ onReady }: Pick<HarnessProps, "onReady">): ReactElement => {
  // TODO: should maybe be `Import.use`?
  const run = Import.useImport();
  useEffect(() => onReady(run), [onReady, run]);
  return <span>ready</span>;
};
Inner.displayName = "Inner";

const Harness = ({ onReady, onStatuses }: HarnessProps): ReactElement => (
  <>
    <Inner onReady={onReady} />
    {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
  </>
);
Harness.displayName = "Harness";

describe("useImport", () => {
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
    let run: ((projectKey?: string) => void) | undefined;
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: { project: { version: 0, selected: proj.key } },
    });
    render(<Harness onReady={(r) => (run = r)} />, { wrapper });
    await waitFor(() => expect(run).toBeDefined());
    act(() => run?.());
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
    let run: ((projectKey?: string) => void) | undefined;
    let statuses: Status.NotificationSpec[] = [];
    const { wrapper } = await createConsoleWrapper({
      client: null,
      preloadedState: { project: { version: 0, selected: "project-1" } },
    });
    render(<Harness onReady={(r) => (run = r)} onStatuses={(s) => (statuses = s)} />, {
      wrapper,
    });
    await waitFor(() => expect(run).toBeDefined());
    act(() => run?.());
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.cancel();
    await act(async () => {});
    expect(statuses).toHaveLength(0);
  });
});
