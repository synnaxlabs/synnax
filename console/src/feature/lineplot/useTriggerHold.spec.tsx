// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Panel as PPanel, Triggers } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { client, project } from "@/feature/lineplot/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const renderTriggerHold = async () => {
  const plot = await client.lineplots.create(await project(), {
    name: uniqueName("plot"),
  });
  const tabKey = uuid.create();
  const doc = await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    root: {
      variant: "leaf",
      tabs: [
        {
          variant: "resource",
          key: tabKey,
          resource: lineplot.ontologyID(plot.key),
        },
      ],
    },
  });
  const { wrapper: Console, store } = await createConsoleWrapper({ client });
  store.dispatch(Session.Panel.select({ key: doc.key, windowKey: MAIN_WINDOW }));
  store.dispatch(
    Session.Panel.internalSelectTab({
      key: doc.key,
      tabKey,
      otherTabKeys: [tabKey],
      windowKey: MAIN_WINDOW,
    }),
  );
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Triggers.Provider>{children}</Triggers.Provider>
    </Console>
  );
  const { result } = renderHook(
    () => ({
      hold: LinePlot.useTriggerHold(),
      panel: PPanel.useRetrieve({ key: doc.key }),
    }),
    { wrapper: Wrapper },
  );
  await waitFor(() => expect(result.current.panel.variant).toBe("success"));
  return { store, key: plot.key };
};

describe("lineplot useTriggerHold", () => {
  it("toggles the focused plot's hold when H is pressed", async () => {
    const { store, key } = await renderTriggerHold();
    expect(
      Session.LinePlot.selectControlState({ state: store.getState(), key }).hold,
    ).toBe(false);
    fireEvent.keyDown(window, { code: "KeyH", key: "h" });
    await waitFor(() =>
      expect(
        Session.LinePlot.selectControlState({ state: store.getState(), key }).hold,
      ).toBe(true),
    );
    fireEvent.keyUp(window, { code: "KeyH", key: "h" });
  });
});
