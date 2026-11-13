// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";
import { type ComponentType, type ReactElement } from "react";

import { type Adapter, type SaveParams } from "@/platform/device/adapter";
import { Modals } from "@/platform/modals";
import { type Tree } from "@/platform/tree";

export interface ChangeIdentifierMenuItemProps extends Pick<
  Tree.ContextMenuProps,
  "selection" | "handleError"
> {}

interface InternalProps extends Pick<ChangeIdentifierMenuItemProps, "handleError"> {
  icon: string;
  name: string;
  deviceKey: string;
  identifier: string;
  save: (params: SaveParams) => Promise<boolean>;
}

const Internal = ({
  icon,
  name,
  deviceKey,
  identifier,
  save,
  handleError,
}: InternalProps) => {
  const rename = Modals.useRename();
  const handleClick = () =>
    handleError(async () => {
      try {
        const newIdentifier = await rename({
          initialValue: identifier,
          allowEmpty: false,
          label: "Identifier",
          title: "Device.Identifier",
          icon: Icon.resolve(icon),
        });
        if (newIdentifier == null) return;
        await save({ key: deviceKey, name, identifier: newIdentifier });
      } catch (e) {
        if (e instanceof Error && errors.Canceled.matches(e)) return;
        throw errors.fromUnknown(e);
      }
    }, "Failed to change identifier");
  return (
    <Menu.Item itemKey="changeIdentifier" onClick={handleClick}>
      <Icon.Hardware />
      Change identifier
    </Menu.Item>
  );
};

export interface CreateChangeIdentifierMenuItemArgs {
  adapter: Adapter;
  /** The name of the icon of the identifier modal, such as `Logo.NI`. */
  icon: string;
}

/** Creates the menu item that changes the identifier of a configured device. */
export const createChangeIdentifierMenuItem = ({
  adapter: { use: useConfigurable, useSave, resourceID },
  icon,
}: CreateChangeIdentifierMenuItemArgs): ComponentType<ChangeIdentifierMenuItemProps> => {
  const ChangeIdentifierMenuItem = ({
    selection: { ids },
    handleError,
  }: ChangeIdentifierMenuItemProps): ReactElement | null => {
    const dev = useConfigurable(ids[0].key);
    const { updateAsync } = useSave();
    const hasUpdatePermission = Access.useUpdateGranted(resourceID(ids[0].key));
    if (ids.length !== 1 || !dev.configured || !hasUpdatePermission) return null;
    return (
      <Internal
        icon={icon}
        name={dev.name}
        deviceKey={dev.key}
        identifier={dev.identifier}
        save={updateAsync}
        handleError={handleError}
      />
    );
  };
  return ChangeIdentifierMenuItem;
};
