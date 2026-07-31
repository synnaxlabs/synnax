// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

describe("LinePlot.useLink", () => {
  it("should open the retrieved line plot as a tab", async () => {
    const project = await client.projects.create({ name: id.create(), layout: {} });
    const linePlot = await client.lineplots.create(project.key, {
      name: "Tank Pressure",
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(project) },
    });
    const { result } = renderHook(() => LinePlot.useLink(), { wrapper });
    await act(async () => {
      await result.current({ client, key: linePlot.key });
    });
    await waitFor(async () => {
      const panelKey = Session.Panel.selectSelected(store.getState());
      expect(panelKey).toBeDefined();
      const doc = await client.panels.retrieve(panelKey as string);
      const tab = panel.findTabByResource(doc.root, lineplot.ontologyID(linePlot.key));
      expect(tab).not.toBeNull();
    });
  });
});
