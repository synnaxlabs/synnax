// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Status, Synch, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";

export const useListenForCalculationState = (): void => {
  const addListener = Synch.useAddListener();
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  useAsyncEffect(async () => {
    if (client == null) return;
    const handler = Synch.getFrameHandlerForStateChannel(
      channel.CALCULATION_STATE_CHANNEL_NAME,
      channel.calculationStateZ,
      ({ key, message, variant }) => {
        const baseStatus = { message, variant };
        if (variant !== status.ERROR_VARIANT) {
          addStatus(baseStatus);
          return;
        }
        handleError(async () => {
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
    );
    return addListener({ channels: channel.CALCULATION_STATE_CHANNEL_NAME, handler });
  }, [client, addListener, addStatus, handleError]);
};
