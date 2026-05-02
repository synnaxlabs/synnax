// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Triggers } from "@/triggers";

type Mode = "copy" | "cut" | "paste" | "clear" | "all" | "undo" | "redo" | "default";

const CONFIG: Triggers.ModeConfig<Mode> = {
  all: [["Control", "A"]],
  copy: [["Control", "C"]],
  cut: [["Control", "X"]],
  paste: [["Control", "V"]],
  clear: [["Escape"]],
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};

const FLATTENED_CONFIG = Triggers.flattenConfig(CONFIG);

export interface UseTriggersProps extends Pick<Triggers.UseProps, "region"> {
  onUndo: () => void;
  onRedo: () => void;
  onCopy: (cursor: xy.XY) => void;
  onCut: (cursor: xy.XY) => void;
  onPaste: (cursor: xy.XY) => void;
  onClear: () => void;
  onSelectAll: () => void;
}

export const useTriggers = ({
  onCopy,
  onCut,
  onPaste,
  onClear,
  onSelectAll,
  onUndo,
  onRedo,
  region,
}: UseTriggersProps) => {
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
        if (mode == "cut") return onCut(cursor);
        if (mode == "paste") return onPaste(cursor);
        if (mode == "clear") return onClear();
        if (mode == "all") return onSelectAll();
      },
      [onUndo, onRedo, onCopy, onCut, onPaste, onClear, onSelectAll],
    ),
  });
};
