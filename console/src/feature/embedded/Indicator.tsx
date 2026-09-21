// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Connection } from "@/platform/connection";

/**
 * Shows in the top bar only while the embedded Core cannot be reached, which is the
 * short gap while the Desktop shell restarts it.
 */
export const Indicator = (): ReactElement | null => {
  const { variant } = Synnax.useConnectionStatus();
  if (variant === "success" || variant === "info" || variant === "disabled")
    return null;
  return (
    <Flex.Box x align="center" gap="small" role="status">
      <Connection.Indicator />
      <Text.Text level="small" color={9}>
        Reconnecting
      </Text.Text>
    </Flex.Box>
  );
};
