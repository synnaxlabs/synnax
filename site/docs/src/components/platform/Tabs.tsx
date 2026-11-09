// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Platform } from "@/components/platform/choice";
import { INFO } from "@/components/platform/Platform";
import { Tabs as Base } from "@/components/tabs";

const TABS = INFO.map(({ key, ...p }) => ({ ...p, tabKey: key }));

export type TabsProps = Base.FilterProps<Platform>;

export const Tabs = (props: TabsProps) => (
  <Base.Tabs queryParamKey="platform" tabs={TABS} {...props} />
);
