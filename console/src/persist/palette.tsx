// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { Icon } from "@synnaxlabs/pluto";

import { type Palette } from "@/palette";
import { CLEAR_STATE } from "@/persist/state";

const CLEAR_LOCAL_STORAGE_COMMAND: Palette.Command = {
  key: "clear-local-storage",
  name: "Clear Local Storage",
  icon: <Icon.Close />,
  onSelect: ({ store, confirm, handleError }) =>
    handleError(async () => {
      const res = await confirm({
        message: "Are you sure you want to clear the Console's local storage?",
        description:
          "This will remove all saved console data that is not persisted within a Synnax cluster.",
      });
      if (!res) return;
      store.dispatch(CLEAR_STATE as PayloadAction<any>);
    }, "Failed to clear local storage"),
};

export const COMMANDS = [CLEAR_LOCAL_STORAGE_COMMAND];
