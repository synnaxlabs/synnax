// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import type { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import pluralize from "pluralize";
import type { ReactElement } from "react";

import { Spec } from "@/feature/schematic/symbol/spec";
import { CSS } from "@/platform/css";

const path = Schematic.Symbol.formPath;

// WHOLE_DRAWING is the list key of the whole drawing, which has no part key.
const WHOLE_DRAWING = "";

export interface PartListProps extends Omit<Input.Control<Spec.Target>, "value"> {
  /** value is the picked part, undefined for the whole drawing, or null for none. */
  value: Spec.Target | null;
  onAdd: () => void;
}

interface PartListItemProps extends List.ItemRenderProps<string> {
  // shapeCount is how many shapes the part reaches. The whole drawing has none.
  shapeCount?: number;
  onDelete: (key: string) => void;
}

// countShapes returns how many shapes in drawing the selectors reach. A group counts as
// each shape inside it.
const countShapes = (drawing: Element, selectors: string[]): number =>
  new Set(Schematic.Node.Region.graphics(drawing, selectors)).size;

const PartListItem = ({
  shapeCount,
  onDelete,
  ...props
}: PartListItemProps): ReactElement | null => {
  const { itemKey } = props;
  const part = Form.useFieldValue(path.data.parts[itemKey], { optional: true });
  const looks = Form.useFieldValue(path.data.looks);
  const whole = itemKey === WHOLE_DRAWING;
  if (!whole && part == null) return null;
  const target = whole ? undefined : itemKey;
  const count = looks.filter((l) => l.part === target).length;
  const shapes =
    shapeCount == null ? "Every shape" : pluralize("shape", shapeCount, true);
  return (
    <Select.Item
      {...props}
      justify="between"
      className={CSS.B("schematic-part-list-item")}
    >
      <Flex.Box y gap={0}>
        {part == null ? (
          <Text.Text level="small">Whole drawing</Text.Text>
        ) : (
          <Form.Field
            path={path.data.parts[itemKey].name}
            showLabel={false}
            padHelpText={false}
          >
            {({ onChange, value }) => (
              <Text.MaybeEditable level="small" value={value} onChange={onChange} />
            )}
          </Form.Field>
        )}
        <Text.Text level="small" color={9}>
          {count === 0 ? shapes : `${shapes} · ${pluralize("look", count, true)}`}
        </Text.Text>
      </Flex.Box>
      {part != null && (
        <Button.Button
          onClick={(e) => {
            e.stopPropagation();
            onDelete(itemKey);
          }}
          size="small"
          variant="text"
          tooltip="Remove part"
          reveal
        >
          <Icon.Close />
        </Button.Button>
      )}
    </Select.Item>
  );
};

/** PartList lists the whole drawing and the parts of the symbol, and picks one. */
export const PartList = ({ value, onChange, onAdd }: PartListProps): ReactElement => {
  const parts = Form.useFieldValue(path.data.parts);
  const markup = Form.useFieldValue(path.data.drawing.markup);
  const { get, set } = Form.useContext();
  const drawing = new DOMParser().parseFromString(
    markup,
    "image/svg+xml",
  ).documentElement;
  const shapeCounts = new Map(
    parts.map((p) => [p.key, countShapes(drawing, p.selectors)]),
  );
  const handleDelete = (key: string) => {
    set(path.data, Spec.removePart(get(path.data).value, key));
    if (value === key) onChange(undefined);
  };
  return (
    <Flex.Box y gap={1} className={CSS.B("schematic-part-list")}>
      <Header.Header level="p" bordered={false}>
        <Header.Title level="p" weight={500}>
          Parts
        </Header.Title>
        <Header.Actions>
          <Button.Button onClick={onAdd} size="small" variant="text" tooltip="Add part">
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <Select.Frame
        value={value === null ? undefined : (value ?? WHOLE_DRAWING)}
        onChange={(key: string) => onChange(key === WHOLE_DRAWING ? undefined : key)}
        data={[WHOLE_DRAWING, ...parts.map((p) => p.key)]}
        closeDialogOnSelect={false}
      >
        <List.Scroll y gap={1}>
          <List.Items<string>>
            {({ key, ...rest }) => (
              <PartListItem
                key={key}
                {...rest}
                shapeCount={shapeCounts.get(key)}
                onDelete={handleDelete}
              />
            )}
          </List.Items>
        </List.Scroll>
      </Select.Frame>
    </Flex.Box>
  );
};
