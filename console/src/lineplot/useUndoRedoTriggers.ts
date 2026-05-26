// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot as PLinePlot, Triggers } from "@synnaxlabs/pluto";
import { useCallback } from "react";

type Mode = "undo" | "redo" | "default";

const CONFIG: Triggers.ModeConfig<Mode> = {
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};

const TRIGGERS = Triggers.flattenConfig(CONFIG);

// useUndoRedoTriggers wires Cmd/Ctrl+Z and Cmd/Ctrl+Shift+Z to the line
// plot's undo/redo stack. Triggers fire only while mounted, so each plot
// tab keeps its own history.
export const useUndoRedoTriggers = (key: string): void => {
  const { undo } = PLinePlot.useUndo({ key });
  const { redo } = PLinePlot.useRedo({ key });
  Triggers.use({
    triggers: TRIGGERS,
    loose: true,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        const mode = Triggers.determineMode(CONFIG, triggers);
        if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [undo, redo],
    ),
  });
};
