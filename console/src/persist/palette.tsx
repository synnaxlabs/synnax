// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PayloadAction } from "@reduxjs/toolkit";
import { Icon } from "@synnaxlabs/media";

import { type Command } from "@/palette/Palette";
import { CLEAR_STATE } from "@/persist/state";

export const defineCommand: Command = {
  key: "clear-local-storage",
  name: "Clear Local Storage",
  icon: <Icon.Close />,
  onSelect: ({ store, confirm }) => {
    confirm(
      {
        message: "Are you sure you want to clear the Console's local storage?",
        description:
          "This will remove all saved console data that is not persisted within a Synnax cluster.",
      },
      {
        icon: "Close",
        name: "Clear Local Storage.Confirm",
      },
    ).then((result) => {
      if (!result) return;
      store.dispatch(CLEAR_STATE as PayloadAction<any>);
    });
  },
};

export const COMMANDS = [defineCommand];
