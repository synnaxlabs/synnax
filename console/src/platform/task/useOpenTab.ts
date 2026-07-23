// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type panel, type rack, type task } from "@synnaxlabs/client";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";

export type FormViewParams = {
  deviceKey?: device.Key;
  taskKey?: task.Key;
  rackKey?: rack.Key;
  config?: unknown;
  /**
   * name is the draft task's semantic name, mirrored from the form's name field so the
   * tab title reflects it before the task is persisted. Absent once taskKey is set: the
   * name is then read from the cluster record.
   */
  name?: string;
};

interface OnSelectParams {
  tabKey?: panel.TabKey;
}

// createOpenTab builds the hook that opens a task form tab of the given type. A tabKey
// from the selector opens the form into that tab in place; otherwise it opens in a new
// tab. The returned callback takes optional form params (an existing task key, a device
// key, or an imported config) and is safe to hand to a trigger surface, which invokes
// it with no arguments to open a blank form.
export const createOpenTab =
  (type: string) =>
  ({ tabKey }: OnSelectParams = {}): ((params?: FormViewParams) => void) => {
    const openTab = Panel.useOpenTab();
    return useCallback(
      (params: FormViewParams = {}) =>
        openTab({ variant: "view", type, args: params, key: tabKey }),
      [openTab, type, tabKey],
    );
  };
