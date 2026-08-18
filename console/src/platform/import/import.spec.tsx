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
import { type Status } from "@synnaxlabs/pluto";
import { act, render, waitFor } from "@testing-library/react";
import { type ReactElement, useEffect } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import {
  CaptureStatuses,
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
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
      client: createTestClient(),
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
