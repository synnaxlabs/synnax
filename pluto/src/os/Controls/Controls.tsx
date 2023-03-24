// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { OS } from "@synnaxlabs/x";

import { MacOSControls } from "./MacOSControls";
import { OSControlsProps } from "./types";
import { WindowsControls } from "./WindowsControls";

import { useOS } from "@/hooks";

const OSControls: Record<OS, React.FC<OSControlsProps>> = {
  MacOS: MacOSControls,
  Windows: WindowsControls,
  Linux: WindowsControls,
  Docker: WindowsControls,
};

const DEFAULT_OS = "Windows";

export interface ControlsProps extends OSControlsProps {
  visibleIfOS?: OS;
}

export const Controls = ({
  forceOS,
  visibleIfOS,
  ...props
}: ControlsProps): JSX.Element | null => {
  const os = useOS(forceOS, DEFAULT_OS) as OS;
  const C = OSControls[os];
  if (visibleIfOS != null && visibleIfOS !== os) return null;
  return <C {...props} />;
};
