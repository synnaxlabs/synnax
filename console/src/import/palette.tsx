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

import { useImport } from "@/import/import";
import { IMPORT_LAYOUT } from "@/import/ImportModal";
import { Layout } from "@/layout";
import { Palette } from "@/palette";

export const ImportCommand: Palette.Command = (listProps) => {
  const handleSelect = useImport();
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import component(s)"
      icon={<Icon.Import />}
      onSelect={handleSelect}
    />
  );
};
ImportCommand.key = "import";
ImportCommand.commandName = "Import component(s)";
ImportCommand.sortOrder = -1;

export const ImportSynnaxCommand: Palette.Command = (listProps) => {
  const placer = Layout.usePlacer();
  const handleSelect = useCallback(() => placer(IMPORT_LAYOUT), [placer]);
  return (
    <Palette.CommandListItem
      {...listProps}
      name="Import Synnax"
      icon={<Icon.Import />}
      onSelect={handleSelect}
    />
  );
};
ImportSynnaxCommand.key = "import-synnax";
ImportSynnaxCommand.commandName = "Import Synnax";

export const COMMANDS = [ImportCommand, ImportSynnaxCommand];
