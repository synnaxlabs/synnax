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
import { deep, id, type payload } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { useIsPreview } from "@/platform/task/useIsPreview";
import { ContextMenu } from "@/platform/task/views/ContextMenu";

type Field = payload.WriteField;

/** Returns base as a field of kind with the zero value of kind. */
const fieldOf = (base: payload.BaseWriteField, kind: payload.Kind): Field => {
  switch (kind) {
    case "float":
      return { ...base, type: kind, value: 0 };
    case "integer":
    case "unsigned":
      return { ...base, type: kind, value: "0" };
    case "boolean":
      return { ...base, type: kind, value: false };
    case "json":
      return { ...base, type: kind, value: "null" };
    case "timestamp":
      return { ...base, type: kind, timeFormat: "iso8601" };
    case "uuid":
      return { ...base, type: kind };
    default:
      return { ...base, type: kind, value: "" };
  }
};

const isGenerated = (kind: payload.Kind): boolean =>
  kind === "uuid" || kind === "timestamp";

const KIND_ITEMS = (
  <>
    <Select.Item<payload.Kind> itemKey="string">String</Select.Item>
    <Select.Item<payload.Kind> itemKey="float">Float</Select.Item>
    <Select.Item<payload.Kind> itemKey="integer">Integer</Select.Item>
    <Select.Item<payload.Kind> itemKey="unsigned">Unsigned integer</Select.Item>
    <Select.Item<payload.Kind> itemKey="boolean">Boolean</Select.Item>
    <Select.Item<payload.Kind> itemKey="json">JSON</Select.Item>
    <Select.Item<payload.Kind> itemKey="bytes">Base64</Select.Item>
    <Select.Item<payload.Kind> itemKey="uuid">UUID</Select.Item>
    <Select.Item<payload.Kind> itemKey="timestamp">Timestamp</Select.Item>
  </>
);

interface ItemProps extends List.ItemProps<string> {
  /** The path of the field. */
  path: deep.Path<Field>;
}

// The section layout dissolves the box of each field, so sizing lives on the inputs.
const POINTER_INPUT_PROPS = { placeholder: "/field", grow: true } as const;

const VALUE_INPUT_PROPS = { className: CSS.B("fixed-field-value") } as const;

const TEXT_INPUT_PROPS = { ...VALUE_INPUT_PROPS, placeholder: "value" } as const;

const INTEGER_INPUT_PROPS = { ...VALUE_INPUT_PROPS, placeholder: "0" } as const;

interface ValueProps {
  path: deep.Path<Field>;
  kind: payload.Kind;
}

const Value = ({ path, kind }: ValueProps): ReactElement | null => {
  switch (kind) {
    case "float":
      return (
        <Form.NumericField
          path={deep.variant(path, "type", "float").value}
          showLabel={false}
          showHelpText={false}
          inputProps={VALUE_INPUT_PROPS}
        />
      );
    case "integer":
    case "unsigned":
      return (
        <Form.TextField
          path={deep.variant(path, "type", kind).value}
          showLabel={false}
          showHelpText={false}
          inputProps={INTEGER_INPUT_PROPS}
        />
      );
    case "boolean":
      return (
        <Form.CheckboxField
          path={deep.variant(path, "type", "boolean").value}
          showLabel={false}
          showHelpText={false}
        />
      );
    case "string":
    case "json":
    case "bytes":
      return (
        <Form.TextField
          path={deep.variant(path, "type", kind).value}
          showLabel={false}
          showHelpText={false}
          inputProps={TEXT_INPUT_PROPS}
        />
      );
    case "timestamp":
      return (
        <Form.Field
          path={deep.variant(path, "type", "timestamp").timeFormat}
          showLabel={false}
          showHelpText={false}
        >
          {({ value, onChange }) => (
            <Select.Simple<payload.TimeFormat>
              value={value}
              onChange={onChange}
              resourceName="time format"
              variant="floating"
            >
              <Select.Item itemKey="iso8601">ISO 8601</Select.Item>
              <Select.Item itemKey="unix_sec">Unix (s)</Select.Item>
              <Select.Item itemKey="unix_ms">Unix (ms)</Select.Item>
              <Select.Item itemKey="unix_us">Unix (µs)</Select.Item>
              <Select.Item itemKey="unix_ns">Unix (ns)</Select.Item>
            </Select.Simple>
          )}
        </Form.Field>
      );
    default:
      return null;
  }
};

const Item = ({ path, ...props }: ItemProps) => {
  const field = Form.useFieldValue(path);
  const { set } = Form.useContext();
  const handleKindChange = (kind: payload.Kind) =>
    set(path, fieldOf({ key: field.key, pointer: field.pointer }, kind));
  return (
    <Select.Item {...props} justify="between" align="center" x>
      <Form.TextField
        path={path.pointer}
        showLabel={false}
        showHelpText={false}
        inputProps={POINTER_INPUT_PROPS}
      />
      <Select.Simple<payload.Kind>
        value={field.type}
        onChange={handleKindChange}
        resourceName="type"
      >
        {KIND_ITEMS}
      </Select.Simple>
      <Value path={path} kind={field.type} />
      <Text.Text level="small" color={9}>
        {isGenerated(field.type) ? "generated" : "fixed"}
      </Text.Text>
    </Select.Item>
  );
};

const EMPTY_CONTENT = <Empty.Action message="No additional fields" />;

export interface AdditionalFieldsProps {
  /** The path of the additional fields of a write target. */
  path: deep.Path<Field[]>;
}

/** Lists the fixed and generated fields that a write adds to its payload. */
export const AdditionalFields = ({ path }: AdditionalFieldsProps): ReactElement => {
  const { data, push, remove } = Form.useFieldList(path);
  const [selected, setSelected] = useState<string[]>([]);
  const isPreview = useIsPreview();
  const add = (kind: payload.Kind) => {
    const field = fieldOf({ key: id.create(), pointer: "" }, kind);
    push(field);
    setSelected([field.key]);
  };
  const handleRemove = (keys: string[]) => {
    remove(keys);
    setSelected([]);
  };
  const listItem = ({ key, ...p }: List.ItemProps<string>) => (
    <Item key={key} {...p} path={path[p.itemKey]} />
  );
  const menuProps = Menu.useContextMenu();
  const menu = ({ keys }: Menu.ContextMenuMenuProps) => (
    <ContextMenu keys={keys} onRemove={handleRemove} />
  );
  const actions = !isPreview && (
    <>
      <Button.Button
        onClick={() => add("string")}
        variant="text"
        size="small"
        tooltip="Add fixed field"
      >
        <Icon.Add />
        Fixed
      </Button.Button>
      <Button.Button
        onClick={() => add("uuid")}
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
