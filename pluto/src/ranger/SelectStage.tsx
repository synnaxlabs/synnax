// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { type Stage, STAGE_ICONS, STAGE_NAMES, STAGES } from "@/ranger/stage";

export interface SelectStageProps extends Omit<
  Select.StaticProps<Stage>,
  "data" | "resourceName"
> {}

export const SelectStage = (props: SelectStageProps): ReactElement => (
  <Select.Static {...props} data={DATA} resourceName="stage" />
);

const DATA: Select.StaticEntry<Stage>[] = STAGES.map((s) => {
  const I = STAGE_ICONS[s];
  return { key: s, name: STAGE_NAMES[s], icon: <I /> };
});
