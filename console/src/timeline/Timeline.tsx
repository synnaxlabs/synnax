// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Icon, Timeline as Core } from "@synnaxlabs/pluto";
import { bounds } from "@synnaxlabs/x";

import { type Layout } from "@/layout";
import { type Selector } from "@/selector";
import { Phase } from "@/timeline/phase";
import { sequenceZ } from "@/timeline/types/types";

export const LAYOUT_TYPE = "timeline";

export const LAYOUT: Layout.State = {
  windowKey: LAYOUT_TYPE,
  key: LAYOUT_TYPE,
  type: LAYOUT_TYPE,
  name: "Timeline",
  location: "mosaic",
  window: {
    title: "Timeline",
  },
};

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Timeline",
  icon: <Icon.Channel />,
  create: async ({ layoutKey }) => ({ ...LAYOUT, key: layoutKey }),
};

export const SELECTABLES: Selector.Selectable[] = [SELECTABLE];

export const Timeline: Layout.Renderer = () => {
  const sequenceForm = Form.use<typeof sequenceZ>({
    schema: sequenceZ,
    values: {
      name: "Sequence 1",
      phases: [
        {
          key: "phase1",
          name: "Phase 1",
          states: [
            {
              key: "state1",
              name: "State 1",
              nodes: [
                {
                  key: "node1",
                  type: "function",
                  config: {},
                },
              ],
            },
          ],
        },
      ],
    },
  });
  return (
    <Form.Form<typeof sequenceZ> {...sequenceForm}>
      <Core.Frame initialBounds={bounds.ZERO}>
        <Phase.Phases />
      </Core.Frame>
    </Form.Form>
  );
};

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Timeline,
};
