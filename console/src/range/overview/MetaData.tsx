// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import {
  Button,
  CSS as PCSS,
  Divider,
  Flex,
  type Flux,
  Form,
  Header,
  Icon,
  Input,
  List,
  Ranger,
  Text,
} from "@synnaxlabs/pluto";
import { type kv, link } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef, useState } from "react";

import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

interface ValueInputProps extends Input.TextProps {}

const ValueInput = ({ value, ...rest }: ValueInputProps): ReactElement => {
  const isLink = link.is(value);
  const copyToClipboard = useCopyToClipboard();
  return (
    <Input.Text
      value={value}
      style={{
        width: "unset",
        flexGrow: 2,
      }}
      selectOnFocus
      variant="shadow"
      resetOnBlurIfEmpty
      placeholder="Value"
      textColor={isLink ? "var(--pluto-primary-z)" : "var(--pluto-gray-l10)"}
      propagateClick
      {...rest}
    >
      <Button.Button onClick={() => copyToClipboard(value, "value")} variant="outlined">
        <Icon.Copy />
      </Button.Button>
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

interface MetaDataListItemProps extends List.ItemProps<string> {
  isCreate?: boolean;
  visible?: boolean;
  rangeKey: ranger.Key;
  onClose?: () => void;
}

const MetaDataListItem = ({
  isCreate = false,
  onClose,
  visible = true,
  rangeKey,
  ...rest
}: MetaDataListItemProps) => {
  const { itemKey } = rest;
  const initialValues = List.useItem<string, ranger.KVPair>(itemKey);
  const inputRef = useRef<HTMLInputElement>(null);
  const { update: handleDelete } = Ranger.useDeleteKV();
  const { form, save } = Ranger.useKVPairForm({
    query: { rangeKey },
    autoSave: !isCreate,
    initialValues: initialValues ?? {
      key: "",
      value: "",
      range: rangeKey,
    },
    sync: !isCreate,
    afterSave: useCallback(
      ({
        reset,
      }: Flux.AfterSaveParams<
        Flux.Shape,
        typeof Ranger.kvPairFormSchema,
        Ranger.FluxSubStore
      >) => {
        onClose?.();
        if (isCreate) reset({ key: "", value: "", range: rangeKey });
      },
      [isCreate, onClose],
    ),
  });
  useEffect(() => {
    if (isCreate) inputRef.current?.focus();
  }, [isCreate, visible]);
  return (
    <List.Item
      className={CSS(
        CSS.BE("metadata", "list-item"),
        isCreate && CSS.M("create"),
        PCSS.visible(visible),
      )}
      preventClick
      propagateClick
      {...rest}
    >
      <Form.Form<typeof Ranger.kvPairFormSchema> {...form}>
        {isCreate ? (
          <Form.TextField
            style={{ flexBasis: "30%", width: 250 }}
            path="key"
            inputProps={{
              ref: inputRef,
              autoFocus: isCreate,
              selectOnFocus: true,
              resetOnBlurIfEmpty: true,
              onlyChangeOnBlur: !isCreate,
              placeholder: "Add Key",
              variant: "shadow",
              weight: 500,
            }}
            showLabel={false}
            hideIfNull
          />
        ) : (
          <Text.Text style={{ flexBasis: "30%", width: 250 }}>
            {initialValues?.key}
          </Text.Text>
        )}
        <Divider.Divider y />
        <Form.Field<string> path="value" showLabel={false} hideIfNull>
          {({ variant: _, ...p }) => <ValueInput onlyChangeOnBlur={!isCreate} {...p} />}
        </Form.Field>
        {isCreate ? (
          <Flex.Box pack>
            <Button.Button
              variant="filled"
              size="small"
              onClick={() => save()}
              trigger={visible ? ["Enter"] : undefined}
            >
              <Icon.Check />
            </Button.Button>
            <Button.Button variant="outlined" size="small" onClick={onClose}>
              <Icon.Close />
            </Button.Button>
          </Flex.Box>
        ) : (
          <Button.Button
            className={CSS.BE("metadata", "delete")}
            size="small"
            variant="shadow"
            onClick={() => handleDelete({ key: itemKey, rangeKey })}
          >
            <Icon.Delete color={10} />
          </Button.Button>
        )}
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
      <Header.Header level="h4" borderColor={5}>
        <Header.Title>Metadata</Header.Title>
        <Header.Actions>
          <Button.Button variant="text" onClick={() => setNewFormVisible(true)}>
            <Icon.Add />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      <List.Frame<string, kv.Pair> data={data} getItem={getItem} subscribe={subscribe}>
        <MetaDataListItem
          key="new"
          index={0}
          itemKey=""
          rangeKey={rangeKey}
          isCreate
          visible={newFormVisible}
          onClose={() => setNewFormVisible(false)}
        />
        <List.Items<string, kv.Pair>>
          {({ key, ...rest }) => (
            <MetaDataListItem key={key} rangeKey={rangeKey} {...rest} />
          )}
        </List.Items>
      </List.Frame>
    </Flex.Box>
  );
};
