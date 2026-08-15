// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul, Icon, Text } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Mosaic } from "@/app/mosaic";
import { Import } from "@/platform/import";
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
import { renderSuspended } from "@/testutil";

const client = createTestClient();

const SeedContent: Panel.Content = () => <Text.Text>seed content</Text.Text>;
const SeedName: Panel.TabName = () => <Text.Text>seed</Text.Text>;
const SeedIcon: Panel.TabIcon = () => <Icon.Visualize />;

const REGISTRY: Panel.Tabs = {
  seed: { Content: SeedContent, Name: SeedName, Icon: SeedIcon },
};

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
    const log = vi.fn();
    const Harness = ({ children }: PropsWithChildren): ReactElement => (
      <Console>
        <Import.FileIngestersProvider fileIngesters={{ log }}>
          <Panel.RendererContext value={REGISTRY}>
            <Haul.Provider>{children}</Haul.Provider>
          </Panel.RendererContext>
        </Import.FileIngestersProvider>
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
    fireFileDrop(getMosaicLeaf(), [
      fakeFileEntry(createJSONFile("widget.json", { type: "log", key: "abc" })),
    ]);
    await waitFor(() => expect(log).toHaveBeenCalledTimes(1));
    expect(log.mock.calls[0][0]).toEqual({ type: "log", key: "abc" });
  });
});
