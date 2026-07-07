// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Panel } from "@/platform/panel";
import { type FormViewArgs } from "@/platform/task/Form";

// createOpenView builds the hook that opens a task form view of the given type. The
// returned callback takes optional args (an existing task key, a device key, or an
// imported config) and is safe to hand to a trigger surface, which invokes it with no
// arguments to open a blank form.
export const createOpenView =
  (type: string) =>
  (): ((args?: FormViewArgs) => void) => {
    const openTab = Panel.useOpenTab();
    return useCallback(
      (args: FormViewArgs = {}) => openTab({ variant: "view", type, args }),
      [openTab],
    );
  };
