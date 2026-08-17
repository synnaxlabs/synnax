// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Empty } from "@/platform/empty";

export const NoneSelected = (): ReactElement => (
  <Text.Text center color={9}>
    No device selected
  </Text.Text>
);

export interface UnconfiguredProps {
  device: Pick<device.Device, "key" | "name">;
  canConfigure: boolean;
  onConfigure: (key: device.Key) => void;
}

export const Unconfigured = ({
  device: { key, name },
  canConfigure,
  onConfigure,
}: UnconfiguredProps): ReactElement => (
  <Empty.Action
    message={`${name} is not configured`}
    action={canConfigure ? `Configure ${name}` : ""}
    onClick={() => onConfigure(key)}
  />
);
