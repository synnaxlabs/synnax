// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Command } from "@/platform/command";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

const useClear = () => {
  const store = Session.useStore();
  const confirm = Modals.useConfirm();
  const handleError = Status.useErrorHandler();
  return useCallback(
    () =>
      handleError(async () => {
        const res = await confirm({
          message: "Are you sure you want to clear the Console's local storage?",
          description:
            "This will remove all saved Console data that is not persisted within a Synnax Core.",
        });
        if (!res) return;
        store.dispatch(Session.Persist.clearState());
      }, "Failed to clear local storage"),
    [store, confirm, handleError],
  );
};

export const ClearCommand = Command.create({
  key: "clear_local_storage",
  name: "Clear local storage",
  icon: <Icon.Close />,
  useOnSelect: useClear,
});

export const COMMANDS = [ClearCommand];
