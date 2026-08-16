// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/platform/panel";
import {
  CaptureStatuses,
  createConsoleWrapper,
  resolveFocusedTab,
  selectTestProject,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

describe("Panel.useOpenResource", () => {
  it("opens the resource's ontology ID as a tab", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const project = await selectTestProject(store, client);
    const created = await client.tables.create(project, { name: uniqueName("table") });
    const { result } = renderHook(() => Panel.useOpenResource(), { wrapper });
    const id = table.ontologyID(created.key);
    await act(async () => {
      result.current(ontology.resourceZ.parse({ id, name: created.name }));
    });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource).toEqual(id);
  });

  it("opens the tab without retrieving the resource", async () => {
    const { wrapper: Wrapper, store } = await createConsoleWrapper({ client });
    await selectTestProject(store, client);
    let statuses: Status.NotificationSpec[] = [];
    const wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Wrapper>
        {children}
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </Wrapper>
    );
    const { result } = renderHook(() => Panel.useOpenResource(), { wrapper });
    // The id references nothing on the cluster, so any retrieval fails loudly.
    const id = table.ontologyID(uuid.create());
    await act(async () => {
      result.current(ontology.resourceZ.parse({ id, name: uniqueName("ghost") }));
    });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource).toEqual(id);
    expect(statuses).toEqual([]);
  });
});
