// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/connection/Target.css";

import { Flex, Synnax, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Indicator } from "@/platform/connection/Indicator";
import { STATUS_LABELS } from "@/platform/connection/labels";
import { useHeldChecking } from "@/platform/connection/useHeldChecking";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

/** The active Core on one line: its name, address, and reachability. */
export const Target = (): ReactElement => {
  const { variant, details } = Synnax.useConnectionStatus();
  const checking = useHeldChecking(details.checking);
  const cluster = Session.Cluster.useSelectState();
  const label = STATUS_LABELS[variant];
  return (
    <Flex.Box
      x
      align="center"
      justify="center"
      gap="medium"
      className={CSS.B("connection-target")}
    >
      <Indicator />
      <Text.Text color={10} weight={500} overflow="ellipsis">
        {cluster?.name ?? "Cluster"}
      </Text.Text>
      {cluster != null && (
        <Text.Text color={9} overflow="ellipsis">
          {cluster.host}:{cluster.port}
        </Text.Text>
      )}
      <Text.Text status={variant} className={CSS.BE("connection-target", "status")}>
        <span>{checking ? "Retrying" : label}</span>
        <span className={CSS.M("ghost")} aria-hidden>
          Retrying
        </span>
        <span className={CSS.M("ghost")} aria-hidden>
          {label}
        </span>
      </Text.Text>
    </Flex.Box>
  );
};
