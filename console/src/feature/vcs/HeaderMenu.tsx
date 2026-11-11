// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vcs } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Access } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useConfigurationModal } from "@/feature/vcs/useConfigurationModal";
import { useConnectModal } from "@/feature/vcs/useConnectModal";
import { Range } from "@/platform/range";
import { Vcs } from "@/platform/vcs";

/** Returns a function that picks a configuration and shows the plan to load it. */
export const useLoadConfiguration = (): (() => void) => {
  const pick = useConfigurationModal();
  const openPlan = Vcs.useOpenPlan();
  const handleError = Status.useErrorHandler();
  return () =>
    handleError(async () => {
      const name = await pick({
        title: "Load configuration",
        icon: <Icon.Pull />,
        description:
          "Pick a configuration to load. You review its changes before they replace the live configuration.",
      });
      if (name == null) return;
      openPlan({
        target: { kind: "load", configuration: name },
        title: `Load ${name}`,
        after: vcs.configurationRev(name),
        afterLabel: name,
      });
    }, "Failed to load configuration");
};

/** Returns a function that picks a configuration to compare the current one with. */
export const useCompareConfigurations = (): (() => void) => {
  const pick = useConfigurationModal();
  const openCompare = Vcs.useOpenCompare();
  const { status } = Vcs.useStatus();
  const handleError = Status.useErrorHandler();
  return () =>
    handleError(async () => {
      const current = status?.configuration ?? "";
      const name = await pick({
        title: "Compare configurations",
        icon: <Icon.Compare />,
        description: `Pick a configuration to compare with ${current}.`,
      });
      if (name == null) return;
      openCompare({
        title: `Compare with ${name}`,
        from: vcs.configurationRev(name),
        fromLabel: name,
        to: vcs.HEAD,
        toLabel: current,
      });
    }, "Failed to compare configurations");
};

const Items = (): ReactElement => {
  const { close } = Dialog.useContext();
  const openConnect = useConnectModal();
  const promptClone = Vcs.useCloneModal();
  const load = useLoadConfiguration();
  const compare = useCompareConfigurations();
  const snapshot = Vcs.useSnapshot();
  const snapshotAllowed = Vcs.useSnapshotAllowed();
  const handleError = Status.useErrorHandler();
  const canConnect = Access.useDeleteGranted(Vcs.RESOURCE_ID);
  const canUpdate = Access.useUpdateGranted(Vcs.RESOURCE_ID);
  const activeRange = Range.useResolve();
  const range =
    activeRange?.variant === "persisted"
      ? { key: activeRange.key, name: activeRange.name }
      : null;
  const handlers: Record<string, () => void> = {
    connection: () => openConnect(),
    clone: () =>
      handleError(async () => {
        await promptClone();
      }, "Failed to create configuration"),
    load,
    compare,
    snapshot: () => {
      if (range != null) snapshot(range);
    },
  };
  const handleChange = (key: string) => {
    close();
    handlers[key]();
  };
  return (
    <Menu.Menu level="small" gap="small" onChange={handleChange}>
      {canUpdate && (
        <>
          <Menu.Item itemKey="clone">
            <Icon.Branch />
            Clone configuration
          </Menu.Item>
          <Menu.Item itemKey="load">
            <Icon.Pull />
            Load configuration
          </Menu.Item>
        </>
      )}
      <Menu.Item itemKey="compare">
        <Icon.Compare />
        Compare configurations
      </Menu.Item>
      {snapshotAllowed && range != null && (
        <>
          <Menu.Divider />
          <Menu.Item itemKey="snapshot">
            <Icon.Snapshot />
            Snapshot configuration to {range.name}
          </Menu.Item>
        </>
      )}
      {canConnect && (
        <>
          <Menu.Divider />
          <Menu.Item itemKey="connection">
            <Icon.Settings />
            Connection settings
          </Menu.Item>
        </>
      )}
    </Menu.Menu>
  );
};

/** The menu in the header of the version control toolbar. */
export const HeaderMenu = (): ReactElement => (
  <Dialog.Frame location={location.BOTTOM_RIGHT}>
    <Dialog.Trigger
      hideCaret
      variant="text"
      size="small"
      rounded="small"
      textColor={9}
      full={false}
      tooltip="More actions"
      aria-label="More version control actions"
    >
      <Icon.KebabMenu />
    </Dialog.Trigger>
    <Dialog.Dialog background={1} pack={false} bordered>
      <Items />
    </Dialog.Dialog>
  </Dialog.Frame>
);
