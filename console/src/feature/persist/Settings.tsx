// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { type ReactElement } from "react";

import { useClear } from "@/feature/persist/useClear";
import { Settings as PlatformSettings } from "@/platform/settings";

/** The settings for the Console data saved on this computer. */
export const Settings = (): ReactElement => {
  const clear = useClear();
  return (
    <PlatformSettings.Card title="Local storage">
      <PlatformSettings.Entry
        label="Clear local storage"
        description="Removes all saved Console data that is not stored in a Core."
      >
        <Button.Button variant="outlined" status="error" onClick={clear}>
          Clear
        </Button.Button>
      </PlatformSettings.Entry>
    </PlatformSettings.Card>
  );
};
