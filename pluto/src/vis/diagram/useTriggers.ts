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

type Mode = "copy" | "paste" | "clear" | "all" | "undo" | "redo" | "default";

const CONFIG: Triggers.ModeConfig<Mode> = {
  defaultMode: "default",
  modes: {
    all: [["Control", "A"]],
    copy: [["Control", "C"]],
    paste: [["Control", "V"]],
    clear: [Triggers.ESCAPE],
    undo: [Triggers.UNDO],
    redo: [Triggers.REDO],
    default: [],
  },
};

const FLATTENED_CONFIG = Triggers.flattenConfig(CONFIG);

const MUTATING_MODES = new Set<Mode>(["undo", "redo", "paste"]);

export interface UseTriggersProps {
  onUndo?: () => void;
  onRedo?: () => void;
  onCopy?: (cursor: xy.XY) => void;
  onPaste?: (cursor: xy.XY) => void;
  onClearSelection?: () => void;
  onSelectAll?: () => void;
  enabled?: Triggers.Condition;
  /** Withholds the shortcuts that change the diagram. Copying and the selection
   * shortcuts stay live, so a read-only diagram is still navigable. Defaults to true.
   * */
  editable?: boolean;
}

export const useTriggers = ({
  onCopy,
  onPaste,
  onClearSelection: onClear,
  onSelectAll,
  onUndo,
  onRedo,
  enabled,
  editable = true,
}: UseTriggersProps) => {
  Triggers.use({
    triggers: FLATTENED_CONFIG,
    loose: true,
    enabled,
    callback: useCallback(
      ({ triggers, cursor, stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        const mode = Triggers.determineMode(CONFIG, triggers);
        if (!editable && MUTATING_MODES.has(mode)) return;
        if (mode == "undo") return onUndo?.();
        if (mode == "redo") return onRedo?.();
        if (mode == "copy") return onCopy?.(cursor);
        if (mode == "paste") return onPaste?.(cursor);
        if (mode == "clear") return onClear?.();
        if (mode == "all") return onSelectAll?.();
      },
      [onUndo, onRedo, onCopy, onPaste, onClear, onSelectAll, editable],
    ),
  });
};
