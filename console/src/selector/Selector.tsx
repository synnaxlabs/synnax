// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/selector/Selector.css";

import { Button, Eraser, Flex, type Icon, Status, Text } from "@synnaxlabs/pluto";
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
  useVisible?: () => boolean;
  create: (props: SelectableCreateArgs) => Promise<Layout.PlacerArgs | null>;
}

export interface SelectorProps extends Layout.RendererProps {
  text: string;
  selectables: Selectable[];
}

interface SelectableItemProps {
  item: Selectable;
  layoutKey: string;
  rename: Modals.PromptRename;
  onPlace: (args: Layout.PlacerArgs) => void;
  onError: (err: unknown, message: string) => void;
}

const SelectableItem = ({
  item,
  layoutKey,
  rename,
  onPlace,
  onError,
}: SelectableItemProps): ReactElement | null => {
  const isVisible = item.useVisible?.() ?? true;
  if (!isVisible) return null;

  const { key, title, icon, create } = item;

  return (
    <Button.Button
      key={key}
      variant="outlined"
      onClick={() =>
        onError(async () => {
          const layout = await create({ layoutKey, rename });
          if (layout != null) onPlace(layout);
        }, `Failed to create ${title}`)
      }
      style={{ flexBasis: "185px" }}
    >
      {icon}
      {title}
    </Button.Button>
  );
};

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
      <Flex.Box className={CSS.B("vis-layout-selector")} gap="large" wrap center>
        <Text.Text level="h4" color={10} weight={400}>
          {text}
        </Text.Text>
        <Flex.Box
          x
          wrap
          style={{ maxWidth: "500px" }}
          full="x"
          justify="center"
          gap={2.5}
        >
          {selectables.map((item) => (
            <SelectableItem
              key={item.key}
              item={item}
              layoutKey={layoutKey}
              rename={rename}
              onPlace={place}
              onError={handleError}
            />
          ))}
        </Flex.Box>
      </Flex.Box>
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
