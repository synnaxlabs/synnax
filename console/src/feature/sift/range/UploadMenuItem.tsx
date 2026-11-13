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

import {
  type UploadModalParams,
  useUploadModal,
} from "@/feature/sift/range/useUploadModal";
import { FLAGS } from "@/flags";

export interface UploadMenuItemProps extends UploadModalParams {}

const Item = (props: UploadMenuItemProps) => {
  const open = useUploadModal();
  return (
    <Menu.Item itemKey="siftUpload" onClick={() => open(props)}>
      <Icon.Logo.Sift />
      Upload to Sift
    </Menu.Item>
  );
};

const Hidden = (): null => null;

/** Opens a modal that uploads the data of a range to Sift. */
export const UploadMenuItem = FLAGS.sift ? Item : Hidden;
