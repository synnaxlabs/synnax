// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { unique } from "@synnaxlabs/x";
import { type ReactElement, useEffect } from "react";

import { Aether } from "@/aether";
import { control } from "@/telem/control/aether";
import { useContext } from "@/telem/control/Controller";
import { Legend as Base } from "@/vis/legend";

export interface LegendProps extends Base.SimpleProps {}

export const Legend = (props: LegendProps): ReactElement => {
  const { needsControlOf } = useContext();
  const [, { states }, setState] = Aether.use({
    type: control.Legend.TYPE,
    schema: control.legendStateZ,
    initialState: { states: [], needsControlOf },
  });

  useEffect(() => {
    setState((state) => ({ ...state, needsControlOf }));
  }, [needsControlOf]);

  // Filter out the unique subjects
  const subjects = unique.unique(states.map((s) => s.subject.key));
  const data = subjects.map((s) => {
    const d = states.find((s2) => s2.subject.key === s);
    if (d == null) throw new UnexpectedError("Legend subject not found");
    return { key: d.subject.key, label: d.subject.name, color: d.subjectColor };
  });

  return <Base.Simple data={data} {...props} />;
};
