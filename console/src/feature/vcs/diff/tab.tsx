// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { TAB_TYPE, useOpened } from "@/feature/vcs/diff/Provider";
import { Panel } from "@/platform/panel";

const Name: Panel.TabName = ({ level, weight, color }): ReactElement => {
  const opened = useOpened();
  return (
    <>
      <Icon.Diff />
      <Text.Text overflow="fade" level={level} weight={weight} color={color}>
        {opened == null ? "Changes" : `${opened.target.name} changes`}
      </Text.Text>
    </>
  );
};

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/vcs/diff/Diff"), "Diff"),
  Icon: Icon.Diff,
  Name,
};

export const TABS: Panel.Tabs = { [TAB_TYPE]: TAB };
