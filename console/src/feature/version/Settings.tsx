// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Settings as PlatformSettings } from "@/platform/settings";
import { Version } from "@/platform/version";
import { Session } from "@/session";

/** The Console version and, on Desktop, its updates. */
export const Settings = (): ReactElement => {
  const version = Session.Version.use();
  return (
    <PlatformSettings.Card title="Console">
      {version != null && (
        <PlatformSettings.Entry label="Version">
          <Text.Text color={10}>{`v${version}`}</Text.Text>
        </PlatformSettings.Entry>
      )}
      {Session.Runtime.ENGINE === "tauri" && (
        <PlatformSettings.Entry label="Updates">
          <Flex.Box x align="center" gap="medium">
            <Version.Update level="p" />
          </Flex.Box>
        </PlatformSettings.Entry>
      )}
    </PlatformSettings.Card>
  );
};
