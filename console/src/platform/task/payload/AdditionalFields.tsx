// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/task/payload/AdditionalFields.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { id, json } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { useIsPreview } from "@/platform/task/Form";
import { JSON_TYPE_ITEMS } from "@/platform/task/payload/JSONTypeItems";
import {
  type GeneratorType,
  type JSONType,
  type TimeFormat,
} from "@/platform/task/payload/types";
import { ContextMenu } from "@/platform/task/views/ContextMenu";

interface StaticField {
  key: string;
  pointer: string;
  type: "static";
  jsonType: JSONType;
  value: unknown;
}

interface GeneratedField {
  key: string;
  pointer: string;
  type: "generated";
  generator: GeneratorType;
  timeFormat?: TimeFormat;
}

type Field = StaticField | GeneratedField;

interface ItemProps extends List.ItemProps<string> {
  /** The path of the field. */
  path: string;
}

// The section layout dissolves the box of each field, so sizing lives on the inputs.
const POINTER_INPUT_PROPS = { placeholder: "/field", grow: true } as const;

const STRING_INPUT_PROPS = {
  placeholder: "value",
  className: CSS.B("static-field-value"),
} as const;

const NUMBER_INPUT_PROPS = { className: CSS.B("static-field-value") } as const;

const PointerField = ({ path }: Pick<ItemProps, "path">) => (
  <Form.TextField
    path={`${path}.pointer`}
    showLabel={false}
    showHelpText={false}
    inputProps={POINTER_INPUT_PROPS}
  />
);

const StaticItem = ({ path, ...props }: ItemProps) => {
  const jsonType = Form.useFieldValue<JSONType>(`${path}.jsonType`);
  const { set } = Form.useContext();
  const handleJSONTypeChange = (value: JSONType) => {
    set(`${path}.jsonType`, value);
    set(`${path}.value`, json.ZERO_PRIMITIVES[value]);
  };
  return (
    <Select.Item {...props} justify="between" align="center" x>
      <PointerField path={path} />
      <Select.Simple<JSONType>
        value={jsonType}
        onChange={handleJSONTypeChange}
        resourceName="type"
      >
        {JSON_TYPE_ITEMS}
      </Select.Simple>
      {jsonType === "string" && (
        <Form.TextField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={STRING_INPUT_PROPS}
        />
      )}
      {jsonType === "number" && (
        <Form.NumericField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
          inputProps={NUMBER_INPUT_PROPS}
        />
      )}
      {jsonType === "boolean" && (
        <Form.SwitchField
          path={`${path}.value`}
          showLabel={false}
          showHelpText={false}
        />
      )}
      <Text.Text level="small" color={9}>
        static
      </Text.Text>
    </Select.Item>
  );
};

type GeneratorKey = "uuid" | TimeFormat;

/** A timestamp generator shows as its time format; ISO 8601 when unset. */
const generatorKey = (
  generator: GeneratorType,
  timeFormat: TimeFormat | undefined,
): GeneratorKey => (generator === "timestamp" ? (timeFormat ?? "iso8601") : "uuid");

const GeneratedItem = ({ path, ...props }: ItemProps) => {
  const generator = Form.useFieldValue<GeneratorType>(`${path}.generator`);
  const timeFormat = Form.useFieldValue<TimeFormat | undefined>(`${path}.timeFormat`);
  const { set } = Form.useContext();
  const handleChange = (key: GeneratorKey) => {
    if (key === "uuid") {
      set(`${path}.generator`, "uuid");
      set(`${path}.timeFormat`, undefined);
    } else {
      set(`${path}.generator`, "timestamp");
      set(`${path}.timeFormat`, key);
    }
  };
  return (
    <Select.Item {...props} justify="between" align="center" x>
      <PointerField path={path} />
      <Select.Simple<GeneratorKey>
        value={generatorKey(generator, timeFormat ?? undefined)}
        onChange={handleChange}
        resourceName="generator"
        variant="floating"
      >
        <Select.Item itemKey="uuid">UUID</Select.Item>
        <Select.Item itemKey="iso8601">Timestamp (ISO 8601)</Select.Item>
        <Select.Item itemKey="unix_sec">Timestamp (s)</Select.Item>
        <Select.Item itemKey="unix_ms">Timestamp (ms)</Select.Item>
        <Select.Item itemKey="unix_us">Timestamp (µs)</Select.Item>
        <Select.Item itemKey="unix_ns">Timestamp (ns)</Select.Item>
      </Select.Simple>
      <Text.Text level="small" color={9}>
        generated
      </Text.Text>
    </Select.Item>
  );
};

const Item = (props: ItemProps) => {
  const type = Form.useFieldValue<Field["type"]>(`${props.path}.type`);
  return type === "static" ? <StaticItem {...props} /> : <GeneratedItem {...props} />;
};

const EMPTY_CONTENT = <Empty.Action message="No additional fields" />;

export interface AdditionalFieldsProps {
  /** The path of the additional fields of a write target. */
  path: string;
}

/** Lists the static and generated fields that a write adds to its payload. */
export const AdditionalFields = ({ path }: AdditionalFieldsProps): ReactElement => {
  const { data, push, remove } = Form.useFieldList<string, Field>(path);
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = useIsPreview();
  const add = (field: Field) => {
    push(field);
    setSelected([field.key]);
  };
  const handleAddStatic = () =>
    add({
      key: id.create(),
      pointer: "",
      type: "static",
      jsonType: "string",
      value: "",
    });
  const handleAddGenerated = () =>
    add({ key: id.create(), pointer: "", type: "generated", generator: "uuid" });
  const handleRemove = (keys: string[]) => {
    remove(keys);
    setSelected([]);
  };
  const listItem = ({ key, ...p }: List.ItemProps<string>) => (
    <Item key={key} {...p} path={`${path}.${p.itemKey}`} />
  );
  const menuProps = Menu.useContextMenu();
  const menu = ({ keys }: Menu.ContextMenuMenuProps) => (
    <ContextMenu keys={keys} onRemove={handleRemove} />
  );
  const actions = !isPreview && (
    <>
      <Button.Button
        onClick={handleAddStatic}
        variant="text"
        size="small"
        tooltip="Add static field"
      >
        <Icon.Add />
        Static
      </Button.Button>
      <Button.Button
        onClick={handleAddGenerated}
        variant="text"
        size="small"
        tooltip="Add generated field"
      >
        <Icon.Time />
        Generated
      </Button.Button>
    </>
  );
  return (
    <Form.Section title="Additional fields" actions={actions}>
      <Menu.ContextMenu {...menuProps} menu={menu}>
        <Select.Frame<string, Field>
          multiple
          data={data}
          value={selected}
          onChange={setSelected}
          replaceOnSingle
          allowNone={false}
          autoSelectOnNone
        >
          <List.Scroll className={menuProps.className} onContextMenu={menuProps.open}>
            <List.Items<string, Field> emptyContent={EMPTY_CONTENT}>
              {listItem}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </Menu.ContextMenu>
    </Form.Section>
  );
};
