// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useAsyncEffect } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useState,
} from "react";

import {
  onStatusChange,
  retrieveStatus,
  STARTING_STATUS,
  type Status,
} from "@/feature/embedded/supervisor";

const Context = createContext<Status>(STARTING_STATUS);
Context.displayName = "Context";

/** @returns The live status of the embedded Core. */
export const useStatus = (): Status => use(Context);

/** Follows the embedded Core of this launch and provides its status. */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const [status, setStatus] = useState<Status>(STARTING_STATUS);
  useAsyncEffect(async (signal) => {
    // An event can land before the first retrieve answers. The event is newer.
    let changed = false;
    const unlisten = await onStatusChange((next) => {
      changed = true;
      if (!signal.aborted) setStatus(next);
    });
    try {
      const current = await retrieveStatus();
      if (!signal.aborted && !changed) setStatus(current);
    } catch (err) {
      unlisten();
      throw errors.fromUnknown(err);
    }
    return unlisten;
  }, []);
  return <Context value={status}>{children}</Context>;
};
