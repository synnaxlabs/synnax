// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { runtime } from "@synnaxlabs/x/runtime";
import type { Flex } from "@synnaxlabs/charon/flex";
import type { Theming } from "@synnaxlabs/charon/theming";

export type ControlsAction = "close" | "minimize" | "maximize";

export interface InternalControlsProps extends Flex.BoxProps {
  forceOS?: runtime.OS;
  disabled?: ControlsAction[];
  focused?: boolean;
  onMinimize?: () => void;
  onMaximize?: () => void;
  onFullscreen?: () => void;
  onClose?: () => void;
  contrast?: Theming.Shade;
}
