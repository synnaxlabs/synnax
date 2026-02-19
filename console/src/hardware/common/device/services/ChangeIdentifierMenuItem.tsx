// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";

import { Modals } from "@/modals";
import { type Ontology } from "@/ontology";

export interface ChangeIdentifierMenuItemProps
  extends Pick<
    Ontology.TreeContextMenuProps,
    "selection" | "state" | "client" | "handleError"
  > {
  icon: string;
}

export const ChangeIdentifierMenuItem = ({
  icon,
  selection: { ids },
  state: { getResource },
  client,
  handleError,
}: ChangeIdentifierMenuItemProps) => {
  const rename = Modals.useRename();
  const first = getResource(ids[0]);
  if (ids.length !== 1 || first.data?.configured !== true) return null;
  const handleClick = () =>
    handleError(async () => {
      const device = await client.devices.retrieve({ key: first.id.key });
      const identifier =
        typeof device.properties.identifier === "string"
          ? device.properties.identifier
          : "";
      try {
        const newIdentifier = await rename(
          { initialValue: identifier, allowEmpty: false, label: "Identifier" },
          { icon, name: "Device.Identifier" },
        );
        if (newIdentifier == null) return;
        await client.devices.create({
          ...device,
          properties: { ...device.properties, identifier: newIdentifier },
        });
      } catch (e) {
        if (e instanceof Error && errors.Canceled.matches(e)) return;
        throw e;
      }
    }, "Failed to change identifier");
  return (
    <Menu.Item itemKey="changeIdentifier" onClick={handleClick}>
      <Icon.Hardware />
      Change identifier
    </Menu.Item>
  );
};
