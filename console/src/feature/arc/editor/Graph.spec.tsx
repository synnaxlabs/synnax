// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { canvas } from "@synnaxlabs/pluto/ether";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { screen } from "@testing-library/react";
import { Suspense } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Editor } from "@/feature/arc/editor";
import { createResourceTab } from "@/platform/panel/testutil";
import { Vis } from "@/platform/vis";
import {
  createConsoleWrapper,
  findIconButton,
  installMissingGPU,
  renderSuspended,
} from "@/testutil";

const client = createTestClient();

const { Content } = Editor.TABS[arc.TYPE_RESOURCE_ID.type];

beforeAll(Content.preload);

describe("Arc editor graph", () => {
  describe("when the canvas has no GPU", () => {
    installMissingGPU();

    it("shows a placeholder in place of the graph and keeps the task controls", async () => {
      const a = await client.arcs.create({
        name: uniqueName("arc"),
        type: "automation",
        source: { mode: "graph" },
      });
      const { wrapper } = await createConsoleWrapper({
        client,
        additionalRegistry: canvas.REGISTRY,
      });
      const { panelKey, tabKey } = await createResourceTab(
        client,
        arc.resourceID(a.key),
      );
      const { container } = await renderSuspended(
        <Vis.Canvas>
          <PlutoPanel.Scope.Provider value={panelKey}>
            <PlutoPanel.TabScope.Provider value={tabKey}>
              <Suspense fallback={null}>
                <Content />
              </Suspense>
            </PlutoPanel.TabScope.Provider>
          </PlutoPanel.Scope.Provider>
        </Vis.Canvas>,
        { wrapper },
      );
      expect(await screen.findByText("GPU required")).toBeDefined();
      expect(
        screen.getByText(
          "Arcs need a GPU to render. Run the Console on a machine with a GPU.",
        ),
      ).toBeDefined();
      expect(await findIconButton(container, "play")).toBeDefined();
    });
  });
});
