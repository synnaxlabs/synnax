// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Channel, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";

export const useListenForCalculationState = (): void => {
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const handleStateUpdate = useCallback(
    ({ key, message, variant }: channel.CalculationState) => {
      const baseStatus = { message, variant };
      if (variant !== "error") {
        addStatus(baseStatus);
        return;
      }
      handleError(async () => {
        if (client == null) throw NULL_CLIENT_ERROR;
        try {
          const { name } = await client.channels.retrieve(key);
          addStatus({
            ...baseStatus,
            description: message,
            message: `Calculation for ${name} failed`,
          });
        } catch {
          addStatus(baseStatus);
        }
      });
    },
    [addStatus, client, handleError],
  );
  Channel.useCalculationStateSynchronizer(handleStateUpdate);
};
