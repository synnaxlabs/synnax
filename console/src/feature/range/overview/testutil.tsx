// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, type project, ranger, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import type { Flux } from "@synnaxlabs/pluto";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act } from "@testing-library/react";
import type { ReactElement } from "react";
import { assert } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import {
  createPanelWrapper,
  createResourceTab,
  primePanel,
} from "@/platform/panel/testutil";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  CaptureStatuses,
  createTestStore,
  renderSuspended,
  type TestStore,
} from "@/testutil";

const client = createTestClient();

const TAB = Range.Overview.TABS[ranger.TYPE_RESOURCE_ID.type];

const Deleted = ({ name }: Flux.Tombstone): ReactElement => <>{name} was deleted</>;

/** The Core that each rendered overview connects to. */
export const CORE = createCore("overview");

export interface RenderOverviewResult {
  store: TestStore;
  project: project.Key;
  /** Points the tab of the overview at the range. */
  setTabResource: (rangeKey: ranger.Key) => Promise<void>;
  /** @returns the status notifications raised so far. */
  statuses: () => Status.NotificationSpec[];
}

export interface RenderOverviewOptions {
  /** The width of the pane that holds the overview, as CSS. */
  width?: string;
}

/** Renders the overview of the range in a tab, as the client `as` sees it. */
export const renderOverview = async (
  rangeKey: ranger.Key,
  as: Client = client,
  { width }: RenderOverviewOptions = {},
): Promise<RenderOverviewResult> => {
  const { panelKey, tabKey } = await createResourceTab(
    client,
    ranger.resourceID(rangeKey),
  );
  const proj = await client.projects.create({ name: uniqueName("proj") });
  const { wrapper, store } = await createPanelWrapper({
    client: as,
    store: await createTestStore({ preloadedState: createCoreState([CORE], CORE.key) }),
    panelKey,
    tabKey,
    project: proj.key,
  });
  await primePanel(wrapper, panelKey);
  const { Content, Guard } = TAB;
  assert(Guard != null);
  let statuses: Status.NotificationSpec[] = [];
  await renderSuspended(
    <>
      <div style={{ width }}>
        <Guard FallbackComponent={Deleted}>
          <Content />
        </Guard>
      </div>
      <Modals.Stack />
      <CaptureStatuses onStatuses={(next) => (statuses = next)} />
    </>,
    { wrapper },
  );
  const setTabResource = async (next: ranger.Key): Promise<void> =>
    await act(async () => {
      await client.panels.dispatch(panelKey, [
        panel.setTabResource({ key: tabKey, resource: ranger.resourceID(next) }),
      ]);
    });
  return { store, project: proj.key, setTabResource, statuses: () => statuses };
};
