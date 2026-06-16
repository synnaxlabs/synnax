// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useNameHook } from "@/layout/context";

export interface TabNameProps extends Omit<Tabs.NameProps, "name" | "onRename"> {
  type: string;
}

export const TabName = (props: TabNameProps): ReactElement => {
  const { tabKey, type } = props;
  const nameProps = useNameHook(type)(tabKey);
  return <Tabs.DefaultName {...nameProps} {...props} />;
};
