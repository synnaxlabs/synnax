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

// Pluto's trigger layer normalizes Meta (Cmd on macOS) into Control before
// dispatching, so Cmd+Z reaches us as ["Control", "Z"].
const UNDO_TRIGGERS: Triggers.Trigger[] = [["Control", "Z"]];
const REDO_TRIGGERS: Triggers.Trigger[] = [["Control", "Shift", "Z"]];

// useUndoRedoTriggers wires Ctrl/Cmd+Z and Ctrl/Cmd+Shift+Z to the Pluto
// undo/redo stack for the given line plot. Triggers fire only when the
// component is mounted, so each plot tab keeps its own undo history.
export const useUndoRedoTriggers = (key: string): void => {
  const { undo } = PLinePlot.useUndo({ key });
  const { redo } = PLinePlot.useRedo({ key });

  Triggers.use({
    triggers: UNDO_TRIGGERS,
    loose: true,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        if (e.stage !== "start") return;
        undo();
      },
      [undo],
    ),
  });

  Triggers.use({
    triggers: REDO_TRIGGERS,
    loose: true,
    callback: useCallback(
      (e: Triggers.UseEvent) => {
        if (e.stage !== "start") return;
        redo();
      },
      [redo],
    ),
  });
};
