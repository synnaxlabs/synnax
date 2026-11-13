// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger, type TimeRange } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";

import { useUploadModal } from "@/feature/sift/range/useUploadModal";
import { FLAGS } from "@/flags";

export interface UploadMenuItemProps {
  rangeKey: ranger.Key;
  rangeName: string;
}

const Item = ({ rangeKey, rangeName }: UploadMenuItemProps) => {
  const open = useUploadModal();
  return (
    <Menu.Item
      itemKey="siftUpload"
      onClick={() => open({ name: rangeName, source: rangeKey })}
    >
      <Icon.Logo.Sift />
      Upload to Sift
    </Menu.Item>
  );
};

export interface UploadRegionMenuItemProps {
  /** The name the modal and its statuses show, such as the name of a plot. */
  name: string;
  /** Resolves the time span to upload, such as a plot selection. */
  getTimeRange: () => Promise<TimeRange>;
}

const RegionItem = ({ name, getTimeRange }: UploadRegionMenuItemProps) => {
  const open = useUploadModal();
  const handleError = Status.useErrorHandler();
  const handleClick = () =>
    handleError(async () => {
      const tr = await getTimeRange();
      open({ name, source: tr.numeric });
    }, "Failed to upload region to Sift");
  return (
    <Menu.Item itemKey="siftUpload" onClick={handleClick}>
      <Icon.Logo.Sift />
      Upload region to Sift
    </Menu.Item>
  );
};

const Hidden = (): null => null;

/** Opens a modal that uploads the data of a range to Sift. */
export const UploadMenuItem = FLAGS.sift ? Item : Hidden;

/** Opens a modal that uploads a plain time span to Sift. */
export const UploadRegionMenuItem = FLAGS.sift ? RegionItem : Hidden;
