// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { Settings } from "@/app/settings/Settings";
import { type Project } from "@/feature/project";
import { Shell } from "@/feature/shell";
import { Session } from "@/session";

/** Settings in place of the project splash, for a session with no project. */
export const Splash = ({ onBack }: Project.SettingsProps): ReactElement => {
  const activeCore = Session.Core.useSelectSelected();
  return (
    <Shell.Frame wide connection={Session.Runtime.CORE_EMBEDDED ? null : activeCore}>
      <Flex.Box y grow empty>
        <Header.Header gap="small" x>
          <Button.Button variant="text" textColor={9} size="medium" onClick={onBack}>
            <Icon.Arrow.Left />
            Projects
          </Button.Button>
        </Header.Header>
        <Settings />
      </Flex.Box>
    </Shell.Frame>
  );
};
