// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List } from "@synnaxlabs/lyra/list";
import { type Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Ranger } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Sift } from "@/feature/sift";
import { Range } from "@/platform/range";

export interface ContextMenuProps
  extends
    Menu.ContextMenuMenuProps,
    Pick<Range.ContextMenuProps, "onStartChild" | "onNext"> {}

/** The range menu for the items of a range list. */
export const ContextMenu = (props: ContextMenuProps): ReactElement => {
  const { keys, target, onStartChild, onNext } = props;
  const single = Ranger.useResult(keys.length === 1 ? { key: keys[0] } : null).data;
  return (
    <Range.ContextMenu
      keys={keys}
      onRename={(r) => Text.edit(List.itemNameID(r.key), target)}
      onStartChild={onStartChild}
      onNext={onNext}
      extraMenuItems={
        single != null && (
          <Sift.Range.UploadMenuItem rangeKey={single.key} rangeName={single.name} />
        )
      }
    />
  );
};
