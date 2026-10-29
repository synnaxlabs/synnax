// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/range/overview/MetaData.css";

import { type ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { type Flux, Ranger } from "@synnaxlabs/pluto";
import { type kv, link } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { CSS } from "@/platform/css";

interface ValueInputProps extends Input.TextProps {}

const ValueInput = ({ value, ...rest }: ValueInputProps): ReactElement => {
  const isLink = link.is(value);
  return (
    <Input.Text
      value={value}
      className={CSS.BE("range-metadata", "value-input")}
      selectOnFocus
      variant="shadow"
      resetOnBlurIfEmpty
      placeholder="Value"
      textColor={isLink ? "var(--pluto-primary-z)" : "var(--pluto-gray-l10)"}
      propagateClick
      {...rest}
    >
      <Button.Copy
        text={value}
        variant="outlined"
        successMessage="Copied value to clipboard"
      />
      {isLink && (
        <Button.Button
          href={value}
          target="_blank"
          autoFormatHref
          variant="outlined"
          propagateClick
        >
          <Icon.LinkExternal />
        </Button.Button>
      )}
    </Input.Text>
  );
};

interface ItemProps extends List.ItemProps<string> {
  rangeKey: ranger.Key;
}

const Item = ({ rangeKey, ...rest }: ItemProps): ReactElement | null => {
  const pair = List.useItem<string, ranger.kv.Pair>(rest.itemKey);
  const { update: handleChange } = Ranger.useUpdateKV();
  const { update: handleDelete } = Ranger.useDeleteKV();
  if (pair == null) return null;
  return (
    <List.Item
      className={CSS.BE("metadata", "list-item")}
      preventClick
      propagateClick
      {...rest}
    >
      <Text.Text className={CSS.BE("range-metadata", "key-cell")}>{pair.key}</Text.Text>
      <Divider.Divider y />
      <Input.Item showLabel={false} padHelpText>
        <ValueInput
          value={pair.value}
          onChange={(value) => handleChange({ ...pair, rangeKey, value })}
          onlyChangeOnBlur
        />
      </Input.Item>
      <Button.Button
        className={CSS.BE("metadata", "delete")}
        size="small"
        variant="text"
        onClick={() => handleDelete({ key: pair.key, rangeKey })}
      >
        <Icon.Delete color={10} />
      </Button.Button>
    </List.Item>
  );
};

interface CreateItemProps {
  rangeKey: ranger.Key;
  visible: boolean;
  onClose: () => void;
}

const CreateItem = ({ rangeKey, visible, onClose }: CreateItemProps): ReactElement => {
  const inputRef = useRef<HTMLInputElement>(null);
  const zero = { key: "", value: "", range: rangeKey };
  const { form, save } = Ranger.useCreateKVPairForm({
    query: null,
    initialValues: zero,
    afterSave: ({
      reset,
    }: Flux.AfterSaveParams<null, typeof Ranger.kvPairFormSchema>) => {
      onClose();
      reset(zero);
    },
  });
  useEffect(() => {
    if (visible) inputRef.current?.focus();
  }, [visible]);
  return (
    <List.Item
      className={CSS.cls(
        CSS.BE("metadata", "list-item"),
        CSS.M("create"),
        PCSS.visible(visible),
      )}
      index={0}
      itemKey=""
      preventClick
      propagateClick
    >
      <Form.Form<typeof Ranger.kvPairFormSchema> {...form}>
        <Form.TextField
          className={CSS.BE("range-metadata", "key-cell")}
          path="key"
          inputProps={{
            ref: inputRef,
            selectOnFocus: true,
            resetOnBlurIfEmpty: true,
            placeholder: "Key",
            variant: "shadow",
            weight: 500,
          }}
          showLabel={false}
        />
        <Divider.Divider y />
        <Form.Field<string> path="value" showLabel={false}>
          {(p) => <ValueInput {...p} />}
        </Form.Field>
        <Flex.Box pack>
          <Button.Button
            variant="filled"
            size="small"
            onClick={() => save()}
            trigger={visible ? ["Enter"] : undefined}
            triggerIndicator
          >
            <Icon.Check />
          </Button.Button>
          <Button.Button variant="outlined" size="small" onClick={onClose}>
            <Icon.Close />
          </Button.Button>
        </Flex.Box>
      </Form.Form>
    </List.Item>
  );
};

export interface MetaDataProps {
  rangeKey: ranger.Key;
}

const sort = (a: kv.Pair, b: kv.Pair) => a.key.localeCompare(b.key);

export const MetaData = ({ rangeKey }: MetaDataProps): ReactElement | null => {
  const [newFormVisible, setNewFormVisible] = useState(false);
  const { data, getItem, subscribe, retrieve, status } = Ranger.useListMetaData({
    initialQuery: { rangeKey },
    sort,
  });
  useEffect(() => retrieve({ rangeKey }), [rangeKey]);
  if (status.variant === "error") return null;
  return (
    <Flex.Box y empty className={CSS.BE("range", "metadata")}>
      <Header.Header level="h4" borderColor={6}>
        <Header.Title>Metadata</Header.Title>
        <Header.Actions>
          <Button.Button variant="text" onClick={() => setNewFormVisible(true)}>
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <List.Frame<string, kv.Pair> data={data} getItem={getItem} subscribe={subscribe}>
        <CreateItem
          rangeKey={rangeKey}
          visible={newFormVisible}
          onClose={() => setNewFormVisible(false)}
        />
        <List.Scroll>
          <List.Items<string, kv.Pair>>
            {({ key, ...rest }) => <Item key={key} rangeKey={rangeKey} {...rest} />}
          </List.Items>
        </List.Scroll>
      </List.Frame>
    </Flex.Box>
  );
};
