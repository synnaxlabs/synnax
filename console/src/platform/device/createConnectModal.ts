// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Rack } from "@synnaxlabs/pluto";

import { getDefaultRack } from "@/platform/device/racks";
import { type ConnectParams } from "@/platform/device/types";
import { Modals } from "@/platform/modals";

/**
 * Returns a prompt hook that opens the connect form of an integration, such as "opc". A
 * caller that names no rack gets {@link getDefaultRack}.
 */
export const createConnectModal = (
  Connect: Modals.Content<ConnectParams, device.Key>,
  integration: string,
): Modals.PromptHook<ConnectParams, device.Key> => {
  const usePrompt = Modals.createPrompt<device.Key, ConnectParams>(Connect);
  const useConnectModal = (): Modals.Prompt<device.Key, ConnectParams> => {
    const prompt = usePrompt();
    const { data: racks } = Rack.useResultMultiple({});
    return (params?: ConnectParams) =>
      prompt({
        ...params,
        rack: params?.rack ?? getDefaultRack(racks ?? [], integration),
      });
  };
  return useConnectModal;
};
