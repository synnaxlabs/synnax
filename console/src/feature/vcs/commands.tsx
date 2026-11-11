// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import {
  useCompareConfigurations,
  useLoadConfiguration,
} from "@/feature/vcs/HeaderMenu";
import { FLAGS } from "@/flags";
import { Command } from "@/platform/command";
import { Range } from "@/platform/range";
import { Vcs } from "@/platform/vcs";

const useEnabled = (): boolean => Vcs.usePolledStatus()?.enabled === true;

const useCanUpdate = (): boolean => {
  const granted = Access.useUpdateGranted(Vcs.RESOURCE_ID);
  return useEnabled() && granted;
};

const useCanSnapshotActiveRange = (): boolean => {
  const allowed = Vcs.useSnapshotAllowed();
  const range = Range.useResolve();
  return allowed && range?.variant === "persisted";
};

const useOpenClone = (): (() => void) => {
  const prompt = Vcs.useCloneModal();
  return () => void prompt();
};

const OpenCommand = Command.create({
  key: "vcs_open",
  name: "Open version control",
  icon: <Icon.SourceControl />,
  useOnSelect: Vcs.useOpenToolbar,
  useVisible: () => Access.useRetrieveGranted(Vcs.RESOURCE_ID),
});

const useSnapshotActiveRange = (): (() => void) => {
  const range = Range.useResolve();
  const snapshot = Vcs.useSnapshot();
  return () => {
    if (range?.variant === "persisted") snapshot({ key: range.key, name: range.name });
  };
};

const SnapshotCommand = Command.create({
  key: "vcs_snapshot_active_range",
  name: "Snapshot configuration to active range",
  icon: <Icon.Snapshot />,
  useOnSelect: useSnapshotActiveRange,
  useVisible: useCanSnapshotActiveRange,
});

const NewConfigurationCommand = Command.create({
  key: "vcs_new_configuration",
  name: "Clone configuration",
  icon: <Icon.Branch />,
  useOnSelect: useOpenClone,
  useVisible: useCanUpdate,
});

const LoadConfigurationCommand = Command.create({
  key: "vcs_load_configuration",
  name: "Load configuration",
  icon: <Icon.Pull />,
  useOnSelect: useLoadConfiguration,
  useVisible: useCanUpdate,
});

const CompareConfigurationsCommand = Command.create({
  key: "vcs_compare_configurations",
  name: "Compare configurations",
  icon: <Icon.Diff />,
  useOnSelect: useCompareConfigurations,
  useVisible: useEnabled,
});

export const COMMANDS: Command.Command[] = FLAGS.versionControl
  ? [
      OpenCommand,
      SnapshotCommand,
      NewConfigurationCommand,
      LoadConfigurationCommand,
      CompareConfigurationsCommand,
    ]
  : [];
