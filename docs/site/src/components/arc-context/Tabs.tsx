// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useLayoutEffect } from "react";

import {
  getFromURL,
  setInURL,
  TABS as ARC_CONTEXTS,
} from "@/components/arc-context/ArcContext";
import { Tabs as Base, type TabsProps as BaseProps } from "@/components/tabs/Tabs";

const TABS = ARC_CONTEXTS.map(({ key, ...c }) => ({ ...c, tabKey: key }));

export type TabsProps = Omit<BaseProps, "tabs" | "queryParamKey">;

export const Tabs = (props: TabsProps) => {
  useLayoutEffect(() => {
    const context = getFromURL();
    if (context) setInURL(context);
  }, []);

  return <Base queryParamKey="context" tabs={TABS} {...props} />;
};
