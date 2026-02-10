// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Palette } from "@/palette";
import { CLEAR_STATE } from "@/persist/state";

export const ClearCommand: Palette.Command = ({
  store,
  confirm,
  handleError,
  ...listProps
}) => {
  const handleSelect = useCallback(
    () =>
      handleError(async () => {
        const res = await confirm({
          message: "Are you sure you want to clear the Console's local storage?",
          description:
            "This will remove all saved Console data that is not persisted within a Synnax Core.",
        });
        if (!res) return;
        store.dispatch(CLEAR_STATE as PayloadAction<any>);
      }, "Failed to clear local storage"),
    [store, confirm, handleError],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Clear local storage"
      icon={<Icon.Close />}
      onSelect={handleSelect}
    />
  );
};
ClearCommand.key = "clear-local-storage";
ClearCommand.commandName = "Clear local storage";

export const COMMANDS = [ClearCommand];
