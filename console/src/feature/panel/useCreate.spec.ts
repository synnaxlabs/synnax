// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/feature/panel";
import { createPanelWrapper } from "@/platform/panel/testutil";
import { Session } from "@/session";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("useCreate", () => {
  it("should create a panel under the selected project and select it", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const { wrapper, store } = await createPanelWrapper({
      client,
      project: proj.key,
    });
    const { result } = renderHook(() => Panel.useCreate(), { wrapper });
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();

    act(() => result.current());
    await waitFor(() =>
      expect(Session.Panel.selectSelected(store.getState())).toBeDefined(),
    );
    const selected = Session.Panel.selectSelected(store.getState());
    if (selected == null) throw new Error("no panel selected after create");

    await waitFor(async () => {
      const doc = await client.panels.retrieve({ key: selected });
      expect(doc.name).toEqual("New Panel");
    });
    const children = await client.panels.retrieve({
      parent: project.ontologyID(proj.key),
    });
    expect(children.map(({ key }) => key)).toContain(selected);
    expect(Session.Panel.selectSelected(store.getState())).toEqual(selected);
  });

  it("should select each subsequently created panel", async () => {
    const { wrapper, store } = await createPanelWrapper({ client });
    const { result } = renderHook(() => Panel.useCreate(), { wrapper });

    act(() => result.current());
    await waitFor(() =>
      expect(Session.Panel.selectSelected(store.getState())).toBeDefined(),
    );
    const first = Session.Panel.selectSelected(store.getState());
    if (first == null) throw new Error("no panel selected after first create");

    act(() => result.current());
    await waitFor(() =>
      expect(Session.Panel.selectSelected(store.getState())).not.toEqual(first),
    );
    const second = Session.Panel.selectSelected(store.getState());
    if (second == null) throw new Error("no panel selected after second create");
    await waitFor(async () => {
      const doc = await client.panels.retrieve({ key: second });
      expect(doc.name).toEqual("New Panel");
    });
  });
});
