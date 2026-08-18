// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul, Icon, Text } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Mosaic } from "@/app/mosaic";
import {
  createJSONFile,
  fakeFileEntry,
  FileDragSource,
  fireFileDrop,
  startFileDrag,
} from "@/platform/import/testutil";
import { Panel } from "@/platform/panel";
import {
  createPanelWrapper,
  createServerPanel,
  getMosaicLeaf,
  primePanel,
} from "@/platform/panel/testutil";
import { assertDefined, renderSuspended } from "@/testutil";

const client = createTestClient();

const SeedContent: Panel.Content = () => <Text.Text>seed content</Text.Text>;
const SeedName: Panel.TabName = () => <Text.Text>seed</Text.Text>;
const SeedIcon: Panel.TabIcon = () => <Icon.Visualize />;
const SEED: Panel.Tab = { Content: SeedContent, Name: SeedName, Icon: SeedIcon };

// The drop opens a resource tab for the log the Core creates, so the registry has to
// render one.
const REGISTRY: Panel.Tabs = { seed: SEED, log: SEED };

const resourceIDs = (node: panel.Node): ontology.ID[] =>
  node.variant === "leaf"
    ? node.tabs.flatMap((tab) => (tab.variant === "resource" ? [tab.resource] : []))
    : [...resourceIDs(node.first), ...resourceIDs(node.last)];

describe("Mosaic file drop", () => {
  it("imports a JSON file dropped on the mosaic", async () => {
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "view", key: uuid.create(), type: "seed", args: {} }],
    });
    const { wrapper: Console } = await createPanelWrapper({
      client,
      panelKey: existing.key,
    });
    await primePanel(Console, existing.key);
    const Harness = ({ children }: PropsWithChildren): ReactElement => (
      <Console>
        <Panel.RendererContext value={REGISTRY}>
          <Haul.Provider>{children}</Haul.Provider>
        </Panel.RendererContext>
      </Console>
    );
    await renderSuspended(
      <>
        <FileDragSource />
        <Mosaic.Mosaic />
      </>,
      { wrapper: Harness },
    );
    await screen.findByText("seed content");
    startFileDrag();
    // A legacy Console log state: the Core names the log it creates after the file.
    fireFileDrop(getMosaicLeaf(), [
      fakeFileEntry(
        createJSONFile("Mosaic Drop.json", {
          version: "0.0.0",
          channels: [1, 2, 3],
          remoteCreated: false,
        }),
      ),
    ]);
    await waitFor(
      async () => {
        const { root } = await client.panels.retrieve(existing.key);
        const [resource] = resourceIDs(root);
        assertDefined(resource, "the drop opened no resource tab");
        const created = await client.logs.retrieve({ key: resource.key });
        expect(created.name).toBe("Mosaic Drop");
      },
      { timeout: 5000 },
    );
  });
});
