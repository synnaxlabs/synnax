// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { Export } from "@/export";
import { LAYOUT_TYPE } from "@/lineplot/layout";
import * as v5 from "@/lineplot/types/v5";

// extract emits a v5 state with the body fields staged in pendingUpload so
// re-import goes through useAutoUpload and lands a fresh plot in the
// destination cluster.
export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const linePlot = await client.lineplots.retrieve({ key });
  const state: v5.State = {
    ...v5.ZERO_STATE,
    key: linePlot.key,
    pendingUpload: {
      title: linePlot.title,
      legend: linePlot.legend,
      channels: linePlot.channels,
      ranges: linePlot.ranges,
      axes: linePlot.axes,
      lines: linePlot.lines,
      rules: linePlot.rules,
    },
  };
  return {
    data: JSON.stringify({ ...state, type: LAYOUT_TYPE }),
    name: linePlot.name,
  };
};

export const useExport = () => Export.use(extract, "line plot");
