// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Provider } from "@synnaxlabs/drift/react";
import { useInitializerRef } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

import { Arc } from "@/session/arc";
import { Cluster } from "@/session/cluster";
import { LinePlot } from "@/session/lineplot";
import { Log } from "@/session/log";
import { Modals } from "@/session/modals";
import { Panel } from "@/session/panel";
import { Project } from "@/session/project";
import { Range } from "@/session/range";
import { Runtime } from "@/session/runtime";
import { Schematic } from "@/session/schematic";
import { Status } from "@/session/status";
import { createStore } from "@/session/store";
import { Synchronizer } from "@/session/synchronizer";
import { Table } from "@/session/table";

export interface ContextProps {
  children: ReactNode;
}

export const Context = ({ children }: ContextProps) => {
  const storeRef = useInitializerRef(() => createStore());
  return (
    <Modals.Context>
      <Provider store={storeRef.current}>{children}</Provider>
    </Modals.Context>
  );
};

const MAIN_SYNCHRONIZERS: Synchronizer.Synchronizers = {
  ...Arc.SYNCHRONIZERS,
  ...Cluster.SYNCHRONIZERS,
  ...LinePlot.SYNCHRONIZERS,
  ...Log.SYNCHRONIZERS,
  ...Panel.SYNCHRONIZERS,
  ...Project.SYNCHRONIZERS,
  ...Range.SYNCHRONIZERS,
  ...Schematic.SYNCHRONIZERS,
  ...Status.SYNCHRONIZERS,
  ...Table.SYNCHRONIZERS,
};

const WINDOW_SYNCHRONIZERS: Synchronizer.Synchronizers = Panel.WINDOW_SYNCHRONIZERS;

/**
 * Mounts the session synchronizers for this window. Must be called below the
 * Pluto providers, in every window.
 */
export const useSynchronizers = (): void =>
  Synchronizer.use(
    Runtime.isMainWindow()
      ? { ...WINDOW_SYNCHRONIZERS, ...MAIN_SYNCHRONIZERS }
      : WINDOW_SYNCHRONIZERS,
  );
