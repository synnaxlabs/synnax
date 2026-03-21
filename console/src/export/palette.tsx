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

import { EXPORT_LAYOUT } from "@/export/ExportModal";
import { Layout } from "@/layout";
import { Palette } from "@/palette";

export const ExportSynnaxCommand: Palette.Command = (listProps) => {
  const placer = Layout.usePlacer();
  const handleSelect = useCallback(() => placer(EXPORT_LAYOUT), [placer]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Export Synnax"
      icon={<Icon.Export />}
      onSelect={handleSelect}
    />
  );
};
ExportSynnaxCommand.key = "export-synnax";
ExportSynnaxCommand.commandName = "Export Synnax";

export const COMMANDS = [ExportSynnaxCommand];
