// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { Flex, Status, Synnax, Text, Tooltip } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

const LABELS: Record<connection.Status["variant"], string> = {
  success: "Connected",
  info: "Connected",
  loading: "Connecting",
  warning: "Reconnecting",
  error: "Error",
  disabled: "Disconnected",
};

export const ConnectionBadge = (): ReactElement => {
  const { variant, message } = Synnax.useConnectionStatus();
  return (
    <Tooltip.Dialog location={location.BOTTOM_LEFT}>
      <Flex.Box y gap="tiny">
        <Text.Text status={variant} weight={650}>
          {LABELS[variant]}
        </Text.Text>
        <Text.Text color={9} weight={450}>
          {message}
        </Text.Text>
      </Flex.Box>
      <Text.Text
        status={variant}
        bordered
        borderColor={6}
        background={variant !== "disabled" && 0}
        size="medium"
        rounded="small"
      >
        <Status.Indicator variant={variant} />
      </Text.Text>
    </Tooltip.Dialog>
  );
};
