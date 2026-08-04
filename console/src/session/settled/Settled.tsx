// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useContext,
} from "react";

import { Cluster } from "@/session/cluster";
import { useSynchronizers } from "@/session/Context";
import { Persist } from "@/session/persist";

const Context = createContext(false);
Context.displayName = "Settled";

/**
 * Runs this window's session synchronizers and publishes whether their first
 * reconcile pass has completed. Mounts once per window, above any boundary
 * that could crash the workspace: repair must outlive what it repairs.
 */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const verified = useSynchronizers();
  return <Context value={verified}>{children}</Context>;
};

/**
 * Whether the workspace state is verified against the connected cluster:
 * first contact made, the session's cluster identity agrees with the
 * connection's, no partition swap is in flight, and a reconcile pass has
 * completed. While false the workspace is in structural doubt and must not
 * render; the connection guard shows a splash instead.
 */
export const use = (): boolean => {
  const verified = useContext(Context);
  const { variant, details } = Synnax.useConnectionStatus();
  const selected = Cluster.useSelectSelectedKey();
  const swapping = Persist.useSelectSwapping();
  // A disabled machine (no client, or a closed or test client) has nothing to
  // verify against; the guards above handle the no-intent case.
  if (variant === "disabled") return true;
  return details.epoch >= 1 && details.clusterKey === selected && !swapping && verified;
};
