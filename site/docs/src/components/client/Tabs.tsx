// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Client } from "@/components/client/choice";
import { INFO } from "@/components/client/Client";
import { Tabs as Base } from "@/components/tabs";

const TABS = INFO.map(({ key, ...c }) => ({ ...c, tabKey: key }));

export type TabsProps = Base.FilterProps<Client>;

export const Tabs = (props: TabsProps) => (
  <Base.Tabs queryParamKey="client" tabs={TABS} {...props} />
);
