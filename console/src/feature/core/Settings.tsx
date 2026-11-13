// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/core/Settings.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Connection } from "@/platform/connection";
import { CSS } from "@/platform/css";
import { Settings as PlatformSettings } from "@/platform/settings";
import { Session } from "@/session";

/** The connected Core: its address, connection status, and version. */
export const Settings = (): ReactElement | null => {
  const { variant, details } = Synnax.useConnectionStatus();
  const core = Session.Core.useSelectSelected();
  if (core == null) return null;
  const address = `${core.host}:${core.port}`;
  return (
    <PlatformSettings.Card title="Core">
      <PlatformSettings.Entry label="Address">
        <Flex.Box x align="center" gap="small">
          <Text.Text color={10} className={CSS.BE("core-settings", "value")}>
            {address}
          </Text.Text>
          <Button.Copy
            variant="text"
            size="small"
            text={address}
            tooltip="Copy address"
          />
        </Flex.Box>
      </PlatformSettings.Entry>
      <PlatformSettings.Entry label="Status">
        <Text.Text status={variant}>{Connection.STATUS_LABELS[variant]}</Text.Text>
      </PlatformSettings.Entry>
      {details.nodeVersion != null && (
        <PlatformSettings.Entry label="Version">
          <Text.Text color={10} className={CSS.BE("core-settings", "value")}>
            {`v${details.nodeVersion}`}
          </Text.Text>
        </PlatformSettings.Entry>
      )}
    </PlatformSettings.Card>
  );
};
