// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { box, type xy } from "@synnaxlabs/x";
import { useCallback } from "react";

type Mode = "copy" | "paste" | "clear" | "all" | "undo" | "redo";

const CONFIG: Triggers.ModeConfig<Mode> = {
  all: [["Control", "A"]],
  copy: [["Control", "C"]],
  paste: [["Control", "V"]],
  clear: [["Escape"]],
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  defaultMode: null,
};

const FLATTENED_CONFIG = Triggers.flattenConfig(CONFIG);

export interface UseUndoProps extends Pick<Triggers.UseProps, "region"> {
  onUndo: () => void;
  onRedo: () => void;
  onCopy: (cursor: xy.XY) => void;
  onPaste: (cursor: xy.XY) => void;
  onClear: () => void;
  onSelectAll: () => void;
}

export const useUndo = ({
  onCopy,
  onPaste,
  onClear,
  onSelectAll,
  onUndo,
  onRedo,
  region,
}: UseUndoProps) => {
  Triggers.use({
    triggers: FLATTENED_CONFIG,
    loose: true,
    region,
    callback: useCallback(
      ({ triggers, cursor, stage }: Triggers.UseEvent) => {
        if (region?.current == null || stage !== "start") return;
        const mode = Triggers.determineMode(CONFIG, triggers);
        if (mode == "undo") return onUndo();
        if (mode == "redo") return onRedo();
        if (mode == "copy") return onCopy(cursor);
        if (mode == "paste") return onPaste(cursor);
        if (mode == "clear") return onClear();
        if (mode == "all") return onSelectAll();
      },
      [onUndo, onRedo, onCopy, onPaste, onClear, onSelectAll],
    ),
  });
};
