// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Selector as AppSelector } from "@/app/selector";
import { Panel } from "@/platform/panel";

export const useSelectorVisible = (): boolean => {
  "use no memo"; // Calls a hook per selectable; VIS_SELECTABLES is a module constant.
  const anySelectable = AppSelector.VIS_SELECTABLES.map(
    // eslint-disable-next-line synnax/react-compiler
    (s) => s.useVisible?.() ?? true,
  ).some(Boolean);
  return Panel.useCanOpenTab() && anySelectable;
};
