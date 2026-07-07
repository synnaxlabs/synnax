// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Log, Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel as PlatformPanel } from "@/platform/panel";

export const TabName: PlatformPanel.TabName = ({ onRename: _, ...props }) => {
  const { key } = Panel.useSelectTabResource();
  Log.useEnsureRetrieved({ key });
  const name = Log.useSelectName({ key });
  const { update } = Log.useRename();
  const handleRename = useCallback(
    (_: string, name: string) => update({ key, name }),
    [update, key],
  );
  return (
    <Panel.DefaultTabName
      {...props}
      icon={<Icon.Log />}
      name={name}
      onRename={handleRename}
    />
  );
};
