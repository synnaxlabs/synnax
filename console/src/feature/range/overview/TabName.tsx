// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Panel as PPanel, Ranger } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel } from "@/platform/panel";

export const TabName: Panel.TabName = ({ onRename: _drop, ...props }) => {
  const { key } = PPanel.useSelectTabResource();
  Ranger.useEnsureRetrieved({ key });
  const name = Ranger.useSelectName({ key });
  const { update } = Ranger.useRename();
  const handleRename = useCallback(
    (_: string, next: string) => update({ key, name: next }),
    [update, key],
  );
  return (
    <PPanel.DefaultTabName
      {...props}
      icon={<Icon.Range />}
      name={name}
      onRename={handleRename}
    />
  );
};
