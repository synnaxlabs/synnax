// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type deep } from "@synnaxlabs/x";
import { useSyncExternalStore } from "react";

import { type Channel, firstPerPort } from "@/platform/task/types";

/**
 * @returns whether deploying the task skips the disabled channel at key, because
 * another channel on its port names the port's data channel. Re-renders the caller
 * only when that changes. Only the channels that binds accepts claim a port.
 */
export const useSkipped = <C extends Channel>(
  path: deep.Target<C[]>,
  key: string,
  portOf: (channel: C) => string,
  binds: (channel: C) => boolean = () => true,
): boolean => {
  const { get, bind } = Form.useContext();
  const getSnapshot = () => {
    const channels = get(path).value.filter(binds);
    const channel = channels.find((c) => c.key === key);
    return (
      channel?.disabled === true && !firstPerPort(channels, portOf).includes(channel)
    );
  };
  return useSyncExternalStore(bind, getSnapshot, getSnapshot);
};
