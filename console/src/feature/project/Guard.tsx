// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

export const Guard = ({ children, Settings }: GuardProps): ReactNode => {
  const selected = Session.Project.useSelectIsAnySelected();
  const [settingsOpen, setSettingsOpen] = useState(false);
  if (selected) return children;
  if (Settings != null && settingsOpen)
    return <Settings onBack={() => setSettingsOpen(false)} />;
  return (
    <Splash
      onOpenSettings={Settings != null ? () => setSettingsOpen(true) : undefined}
    />
  );
};
