// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCreateModal } from "@/agent/CreateModal";
import { create as createEditor } from "@/agent/editor/Editor";
import { Palette } from "@/palette";

export const CreateCommand: Palette.Command = ({
  placeLayout,
  handleError,
  ...listProps
}) => {
  const createModal = useCreateModal();
  const handleSelect = useCallback(
    () =>
      handleError(async () => {
        const result = await createModal({});
        if (result == null) return;
        const { agent } = result;
        placeLayout(
          createEditor({ key: agent.key, name: agent.name }),
        );
      }, "Failed to create Agent"),
    [placeLayout, handleError, createModal],
  );
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Create an Agent"
      icon={<Icon.Auto />}
      onSelect={handleSelect}
    />
  );
};

CreateCommand.key = "agent-create";
CreateCommand.commandName = "Create an Agent";

export const COMMANDS = [CreateCommand];
