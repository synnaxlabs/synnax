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

import { Layout } from "@/layout";
import { MANAGE_CORE_LAYOUT } from "@/manage/ManageCoreModal";
import { Palette } from "@/palette";

export const ManageCoreCommand: Palette.Command = (listProps) => {
  const placer = Layout.usePlacer();
  const handleSelect = useCallback(() => placer(MANAGE_CORE_LAYOUT), [placer]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Manage Core and Data"
      icon={<Icon.Settings />}
      onSelect={handleSelect}
    />
  );
};
ManageCoreCommand.key = "manage-core";
ManageCoreCommand.commandName = "Manage Core and Data";

export const COMMANDS = [ManageCoreCommand];
