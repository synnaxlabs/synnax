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
import {
  type ComponentType,
  type PropsWithChildren,
  type ReactNode,
  useState,
} from "react";

import { Splash } from "@/feature/project/Splash";
import { Session } from "@/session";

/** Props of the Settings view that the splash opens. */
export interface SettingsProps {
  /** Returns to the splash. */
  onBack: () => void;
}

export interface GuardProps extends PropsWithChildren {
  /** Shown in place of the splash when the user opens Settings from it. */
  Settings?: ComponentType<SettingsProps>;
}

type UnselectedProps = Pick<GuardProps, "Settings">;

const Unselected = ({ Settings }: UnselectedProps): ReactNode => {
  const [settingsOpen, setSettingsOpen] = useState(false);
  if (Settings == null) return <Splash />;
  if (settingsOpen) return <Settings onBack={() => setSettingsOpen(false)} />;
  const gear = (
    <Button.Button
      onClick={() => setSettingsOpen(true)}
      variant="text"
      size="medium"
      textColor={9}
      tooltip="Settings"
      aria-label="Settings"
    >
      <Icon.Settings />
    </Button.Button>
  );
  return <Splash actions={gear} />;
};

export const Guard = ({ children, Settings }: GuardProps): ReactNode => {
  const selected = Session.Project.useSelectIsAnySelected();
  if (selected) return children;
  return <Unselected Settings={Settings} />;
};
