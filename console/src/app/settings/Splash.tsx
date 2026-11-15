// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, Suspense } from "react";

import { type Project } from "@/feature/project";
import { Shell } from "@/feature/shell";
import { Panel } from "@/platform/panel";
import { Settings as PlatformSettings } from "@/platform/settings";
import { Session } from "@/session";

const Settings = Panel.lazyComponent(
  () => import("@/app/settings/Settings"),
  "Settings",
);

/** Settings in place of the project splash, for a session with no project. */
export const Splash = ({ onBack }: Project.SettingsProps): ReactElement => {
  const activeCore = Session.Core.useSelectSelected();
  return (
    <Shell.Page
      connection={Session.Runtime.CORE_EMBEDDED ? null : activeCore}
      start={
        <Button.Button variant="text" textColor={9} size="medium" onClick={onBack}>
          <Icon.Arrow.Left />
          Projects
        </Button.Button>
      }
    >
      <PlatformSettings.CardBackgroundContext value={0}>
        <Suspense fallback={null}>
          <Settings />
        </Suspense>
      </PlatformSettings.CardBackgroundContext>
    </Shell.Page>
  );
};
