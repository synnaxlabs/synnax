// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { client, createSchematic, testProjectKey } from "@/feature/schematic/testutil";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

describe("scratch openTab", () => {
  it("opens a tab", async () => {
    const s = await createSchematic({});
    const hook = await renderHookWithConsole(
      () => ({
        openTab: Panel.useOpenTab(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    hook.store.dispatch(Session.Project.select(await testProjectKey()));
    act(() => {
      hook.result.current.openTab({
        variant: "resource",
        resource: schematic.ontologyID(s.key),
      });
    });
    await waitFor(() => {
      console.log(
        "statuses",
        hook.result.current.notifications.statuses.map((st) => [
          st.message,
          st.description,
        ]),
      );
      const panelKey = Session.Panel.selectSelected(hook.store.getState());
      expect(panelKey).not.toBeNull();
      const tabs = Session.Panel.selectSelectedTabs(hook.store.getState(), panelKey!);
      console.log("selected tabs", tabs);
      expect(tabs[0]).toBeDefined();
    });
  });
});
