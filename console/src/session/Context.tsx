// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Provider } from "@synnaxlabs/drift/react";
import { Synnax, useInitializerRef } from "@synnaxlabs/pluto";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useContext,
} from "react";

import { Arc } from "@/session/arc";
import { Cluster } from "@/session/cluster";
import { LinePlot } from "@/session/lineplot";
import { Log } from "@/session/log";
import { Modals } from "@/session/modals";
import { Panel } from "@/session/panel";
import { Persist } from "@/session/persist";
import { Project } from "@/session/project";
import { Range } from "@/session/range";
import { Runtime } from "@/session/runtime";
import { Schematic } from "@/session/schematic";
import { Status } from "@/session/status";
import { type Action, createStore, type State } from "@/session/store";
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

const MAIN_SYNCHRONIZERS: Synchronizer.Synchronizers<State, Action> = {
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

const WINDOW_SYNCHRONIZERS: Synchronizer.Synchronizers<State, Action> =
  Panel.WINDOW_SYNCHRONIZERS;

/**
 * Mounts the session synchronizers for this window. Must be called below the
 * Pluto providers, in every window.
 * @returns whether a full reconcile pass has completed since the last
 * return-to-cold.
 */
const useSynchronizers = (): boolean =>
  Synchronizer.use(
    Runtime.isMainWindow()
      ? { ...WINDOW_SYNCHRONIZERS, ...MAIN_SYNCHRONIZERS }
      : WINDOW_SYNCHRONIZERS,
  );

const SettledContext = createContext(false);
SettledContext.displayName = "Settled";

/**
 * Runs this window's session synchronizers and publishes whether their first
 * reconcile pass has completed. Mounts once per window, above any boundary
 * that could crash the workspace: repair must outlive what it repairs.
 */
export const SettledProvider = ({ children }: PropsWithChildren): ReactElement => {
  const verified = useSynchronizers();
  return <SettledContext value={verified}>{children}</SettledContext>;
};

/**
 * Whether the workspace state is verified against the connected cluster:
 * first contact made, the session's cluster identity agrees with the
 * connection's, no partition swap is in flight, and a reconcile pass has
 * completed. While false the workspace is in structural doubt and must not
 * render; the connection guard shows a splash instead.
 */
export const useSettled = (): boolean => {
  const verified = useContext(SettledContext);
  const { variant, details } = Synnax.useConnectionStatus();
  const selected = Cluster.useSelectSelectedKey();
  const swapping = Persist.useSelectSwapping();
  // A disabled machine (no client, or a closed or test client) has nothing to
  // verify against; the guards above handle the no-intent case.
  if (variant === "disabled") return true;
  return details.epoch >= 1 && details.clusterKey === selected && !swapping && verified;
};
