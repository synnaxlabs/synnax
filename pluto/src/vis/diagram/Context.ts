// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@/context";
import { type Viewport } from "@/viewport";
import { diagram } from "@/vis/diagram/aether";

export interface ContextValue {
  editable: boolean;
  visible: boolean;
  onEditableChange: (v: boolean) => void;
  viewportMode: Viewport.Mode;
  onViewportModeChange: (v: Viewport.Mode) => void;
  fitViewOnResize: boolean;
  setFitViewOnResize: (v: boolean) => void;
  fitViewOptions: diagram.FitViewOptions;
}

export const [Context, useContext] = context.create<ContextValue>({
  defaultValue: {
    editable: true,
    fitViewOnResize: false,
    fitViewOptions: diagram.FIT_VIEW_OPTIONS,
    onEditableChange: () => {},
    onViewportModeChange: () => {},
    setFitViewOnResize: () => {},
    viewportMode: "select",
    visible: true,
  },
  displayName: "Diagram.Context",
});
