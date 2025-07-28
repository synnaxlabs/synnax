// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/selector/Selector.css";

import { Align, Button, Eraser, type Icon, Status, Text } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Modals } from "@/modals";

export interface SelectableCreateArgs {
  layoutKey: string;
  rename: Modals.PromptRename;
}

export interface Selectable {
  key: string;
  title: string;
  icon: Icon.ReactElement;
  create: (props: SelectableCreateArgs) => Promise<Layout.PlacerArgs | null>;
}

export interface SelectorProps extends Layout.RendererProps {
  text: string;
  selectables: Selectable[];
}

export const Selector = ({
  layoutKey,
  selectables,
  text,
}: SelectorProps): ReactElement => {
  const place = Layout.usePlacer();
  const rename = Modals.useRename();
  const handleError = Status.useErrorHandler();
  return (
    <Eraser.Eraser>
      <Align.Center className={CSS.B("vis-layout-selector")} gap="large" wrap>
        <Text.Text level="h4" shade={10} weight={400}>
          {text}
        </Text.Text>
        <Align.Space
          x
          wrap
          style={{ maxWidth: "500px", width: "100%" }}
          justify="center"
          gap={2.5}
        >
          {selectables.map(({ key, title, icon, create }) => (
            <Button.Button
              key={key}
              variant="outlined"
              onClick={() =>
                handleError(async () => {
                  const layout = await create({ layoutKey, rename });
                  if (layout != null) place(layout);
                }, `Failed to create ${title}`)
              }
              startIcon={icon}
              style={{ flexBasis: "185px" }}
            >
              {title}
            </Button.Button>
          ))}
        </Align.Space>
      </Align.Center>
    </Eraser.Eraser>
  );
};

export const createSelector = (
  selectables: Selectable[],
  text: string,
): Layout.Renderer => {
  const C: Layout.Renderer = (props) => (
    <Selector {...props} selectables={selectables} text={text} />
  );
  C.displayName = "LayoutSelector";
  return C;
};
