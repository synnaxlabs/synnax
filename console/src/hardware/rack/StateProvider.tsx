// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { Rack, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useState,
} from "react";

import { StateContext, type States } from "@/hardware/rack/StateContext";

export const StateProvider = (props: PropsWithChildren): ReactElement => {
  const client = Synnax.use();
  const [states, setStates] = useState<States>({});
  useAsyncEffect(async () => {
    if (client == null) return;
    const racks = await client.hardware.racks.retrieve([], { includeState: true });
    const initialStates: States = Object.fromEntries(
      racks
        .filter(({ state }) => state != null)
        .map(({ key, state }) => [key, state as rack.State]),
    );
    setStates(initialStates);
  }, [client]);
  const handleStateUpdate = useCallback((state: rack.State) => {
    setStates((prevStates) => ({ ...prevStates, [state.key]: state }));
  }, []);
  Rack.useStateSynchronizer(handleStateUpdate);
  return <StateContext {...props} value={states} />;
};
