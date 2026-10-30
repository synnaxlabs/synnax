// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type project } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";

import { create, exists, KEYS } from "@/feature/project/sample";
import sequence from "@/feature/project/sampleSequence.arc?raw";
import simulator from "@/feature/project/sampleSimulator.arc?raw";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

// Selecting a project loads its stored view state over the store's, so view state set
// before the load lands is lost.
const load = async (store: Session.Store, key: project.Key): Promise<void> => {
  store.dispatch(Session.Project.select(key));
  const loaded = (): boolean => !Session.Persist.selectSwapping(store.getState());
  if (loaded()) return;
  await new Promise<void>((resolve) => {
    const unsubscribe = store.subscribe(() => {
      if (!loaded()) return;
      unsubscribe();
      resolve();
    });
  });
};

/**
 * Returns a callback that creates the sample project and opens it. When a sample
 * exists, the callback asks before it resets the sample.
 */
export const useOpenSample = (): (() => void) => {
  const client = PSynnax.use();
  const store = Session.useStore();
  const confirm = Modals.useConfirm();
  const handleError = Status.useErrorHandler();
  return () =>
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      if (await exists(client)) {
        const confirmed = await confirm({
          message: "Are you sure you want to reset the sample project?",
          description:
            "Your changes to the sample project and its Arc programs are lost.",
          confirm: { label: "Reset", variant: "warning" },
        });
        if (confirmed !== true) return;
      }
      await create(client, { simulator, sequence });
      await load(store, KEYS.project);
      store.dispatch(
        Session.Schematic.setEditable({ key: KEYS.schematic, editable: false }),
      );
    }, "Failed to open the sample project");
};
