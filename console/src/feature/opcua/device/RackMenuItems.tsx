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
import { type ReactElement } from "react";

import { useDriverCertificateModal } from "@/feature/opcua/device/Certificate";
import { type Device } from "@/platform/device";

export const RackMenuItems = ({ rackKey }: Device.RackMenuItemsProps): ReactElement => {
  const openCertificate = useDriverCertificateModal();
  return (
    <Menu.Item
      itemKey="showOPCUADriverCertificate"
      onClick={() => void openCertificate({ rackKey })}
    >
      <Icon.Logo.OPCUA />
      Show OPC UA Driver certificate
    </Menu.Item>
  );
};
