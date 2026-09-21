// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SynnaxParams } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { useMemo, useState } from "react";

import { useStatus } from "@/feature/embedded/Provider";
import { type Connection } from "@/feature/embedded/supervisor";

/** The name of the embedded Core record. */
export const NAME = "Synnax";

/**
 * @returns The parameters a client needs to reach the embedded Core, or undefined
 * before the Core runs for the first time. The parameters outlive a restart of the
 * Core, so the client keeps its intent to connect and reconnects by itself.
 */
export const useConnParams = (): SynnaxParams | undefined => {
  const status = useStatus();
  const [connection, setConnection] = useState<Connection | null>(null);
  if (status.state === "running" && !deep.equal(connection, status.connection))
    setConnection(status.connection);
  return useMemo(
    () =>
      connection == null ? undefined : { ...connection, name: NAME, secure: false },
    [connection],
  );
};
