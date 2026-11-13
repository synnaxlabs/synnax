// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { OPTIONS } from "@/feature/theme/options";
import { Settings as PlatformSettings } from "@/platform/settings";
import { Session } from "@/session";
import { Theme } from "@/session/theme";

/** The theme settings: light, dark, or the theme of the operating system. */
export const Settings = (): ReactElement => {
  const mode = Theme.useSelectMode();
  const dispatch = Session.useDispatch();
  return (
    <PlatformSettings.Card title="Theme">
      <PlatformSettings.Entry
        label="Color theme"
        description="System follows the light or dark mode of your computer."
      >
        <Select.Buttons<Theme.Mode>
          value={mode}
          onChange={(next: Theme.Mode) => dispatch(Theme.set(next))}
        >
          {OPTIONS.map(({ key, name, icon }) => (
            <Select.Item key={key} itemKey={key}>
              {icon}
              {name}
            </Select.Item>
          ))}
        </Select.Buttons>
      </PlatformSettings.Entry>
    </PlatformSettings.Card>
  );
};
