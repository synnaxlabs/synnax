// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  DataType,
  type Synnax as Client,
  video,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Panel as PlutoPanel, Video } from "@synnaxlabs/pluto";
import { video as aetherVideo, VideoFactory } from "@synnaxlabs/pluto/ether";
import { id } from "@synnaxlabs/x";
import { act, render, within } from "@testing-library/react";
import {
  type ComponentType,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  Suspense,
} from "react";
import { onTestFinished } from "vitest";

import { createResourceTab, primePanel } from "@/platform/panel/testutil";
import { canvasTest, createConsoleWrapper, uniqueName } from "@/testutil";

export const client = createTestClient();

/**
 * Creates a persisted video channel and its index, and deletes both when the test
 * ends.
 */
export const createStoredVideo = async (): Promise<channel.Channel> => {
  const idx = await client.channels.create({
    name: uniqueName("camera_time"),
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  const ch = await client.channels.create({
    name: uniqueName("camera"),
    dataType: DataType.VIDEO,
    index: idx.key,
  });
  onTestFinished(async () => await client.channels.delete([ch.key, idx.key]));
  return ch;
};

let projectKey: string | undefined;
const project = async (): Promise<string> =>
  (projectKey ??= (await client.projects.create({ name: id.create() })).key);

// A single-hook bootstrap primes the Flux cache, since a suspending useEnsure followed
// by other hooks trips a React 19 concurrent-replay error.
const load = async (Wrapper: FC<PropsWithChildren>, key: string): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    Video.useEnsure({ key });
    return <div data-testid="loaded" />;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Suspense fallback={null}>
        <Bootstrap />
      </Suspense>,
      { wrapper: Wrapper },
    );
  });
  await within(utils.container).findByTestId("loaded");
};

export interface RenderVideoOptions {
  video?: Partial<video.New>;
  /** The client that the component renders against. Defaults to the root client. */
  as?: Client;
}

/**
 * Creates a video view on the Core, and mounts Component in the panel and tab scopes
 * of a resource tab for it, the way the mosaic renders a tab.
 */
export const renderVideo = async (
  Component: ComponentType,
  { video: overrides, as = client }: RenderVideoOptions = {},
) => {
  const created = await client.videos.create(await project(), {
    name: "Camera",
    ...overrides,
  });
  // A source with no client reports no pictures, so the view mounts with no Core feed.
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client: as,
    additionalRegistry: aetherVideo.REGISTRY,
    renderContext: canvasTest.record(),
    telemFactories: [new VideoFactory(null)],
  });
  await load(Wrapper, created.key);
  const { panelKey, tabKey } = await createResourceTab(
    client,
    video.ontologyID(created.key),
  );
  await primePanel(Wrapper, panelKey);
  let result!: ReturnType<typeof render>;
  await act(async () => {
    result = render(
      <PlutoPanel.Scope.Provider value={panelKey}>
        <PlutoPanel.TabScope.Provider value={tabKey}>
          <Video.Scope.Provider value={created.key}>
            <Component />
          </Video.Scope.Provider>
        </PlutoPanel.TabScope.Provider>
      </PlutoPanel.Scope.Provider>,
      { wrapper: Wrapper },
    );
  });
  return { key: created.key, result, store, wrapper: Wrapper };
};
