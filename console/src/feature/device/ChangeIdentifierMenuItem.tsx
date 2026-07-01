// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Access, Device, Icon, Menu } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";

import { type Ontology } from "@/feature/ontology";
import { Device as CDevice } from "@/platform/device";
import { Modals } from "@/platform/modals";

export interface ChangeIdentifierMenuItemProps extends Pick<
  Ontology.TreeContextMenuProps,
  "selection" | "state" | "handleError"
> {
  icon: string;
}

export const ChangeIdentifierMenuItem = ({
  icon,
  selection: { ids },
  state: { getResource },
  handleError,
}: ChangeIdentifierMenuItemProps) => {
  const rename = Modals.useRename();
  const { updateAsync } = CDevice.useChangeIdentifier();
  const first = getResource(ids[0]);
  const { data: deviceData } = Device.useRetrieve({ key: first.id.key });
  const hasUpdatePermission = Access.useUpdateGranted(device.ontologyID(ids[0].key));
  if (ids.length !== 1 || first.data?.configured !== true || !hasUpdatePermission)
    return null;
  const identifier =
    typeof deviceData?.properties?.identifier === "string"
      ? deviceData.properties.identifier
      : "";
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
        await updateAsync({ key: first.id.key, identifier: newIdentifier });
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
