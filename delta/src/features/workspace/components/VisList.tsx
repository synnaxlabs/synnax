// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List } from "@synnaxlabs/pluto";

import { RenderableLayout, useSelectLayouts } from "@/features/layout";

export const VisList = (): JSX.Element => {
  const layouts = useSelectLayouts().filter(
    (layout) => layout.type === "visualization"
  );
  return (
    <List<RenderableLayout> data={layouts}>
      <List.Column.Header<RenderableLayout>
        columns={[
          {
            key: "title",
            name: "Title",
          },
        ]}
      />
      <List.Core.Virtual itemHeight={30} style={{ height: "100%" }}>
        {List.Column.Item}
      </List.Core.Virtual>
    </List>
  );
};
