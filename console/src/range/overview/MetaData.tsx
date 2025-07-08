// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Component,
  Divider,
  Icon,
  Input,
  List,
  Ranger,
  Text,
} from "@synnaxlabs/pluto";
import { type kv, link } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

const ValueInput = ({ value, onChange }: Input.Control<string>): ReactElement => {
  const isLink = link.is(value);
  const copyToClipboard = useCopyToClipboard();
  return (
    <Input.Text
      value={value}
      onChange={onChange}
      style={{
        width: "unset",
        flexGrow: 2,
      }}
      variant="shadow"
      selectOnFocus={true}
      resetOnBlurIfEmpty={true}
      onlyChangeOnBlur={true}
      placeholder="Value"
      color={isLink ? "var(--pluto-primary-z)" : "var(--pluto-gray-l10)"}
    >
      <Button.Icon onClick={() => copyToClipboard(value, "value")} variant="outlined">
        <Icon.Copy />
      </Button.Icon>
      {isLink && (
        <Button.Link
          variant="outlined"
          href={value}
          target="_blank"
          autoFormat
          style={{ padding: "1rem" }}
        >
          <Icon.LinkExternal />
        </Button.Link>
      )}
    </Input.Text>
  );
};

const MetaDataListItem = (props: List.ItemProps<string>) => {
  const { itemKey } = props;
  const pair = List.useItem<string, ranger.KVPair>(itemKey);
  if (pair == null) return null;
  const { key, value, range } = pair;
  const update = Ranger.useUpdateKV.useDirect({ params: { rangeKey: range } });
  return (
    <List.Item
      style={{ padding: "0.5rem", border: "none" }}
      className={CSS.BE("metadata", "item")}
      allowSelect={false}
      {...props}
    >
      <Input.Text
        style={{ flexBasis: "30%", width: 250 }}
        variant="shadow"
        value={key}
        selectOnFocus={true}
        resetOnBlurIfEmpty={true}
        onlyChangeOnBlur={true}
        onChange={(value) => update.update({ ...pair, key: value })}
        placeholder="Add Key"
        weight={500}
      />
      <Divider.Divider y />
      {key != null && key.length !== 0 && (
        <>
          <ValueInput
            value={value}
            onChange={(value) => update.update({ ...pair, value })}
          />
          <Button.Icon
            className={CSS.BE("metadata", "delete")}
            size="small"
            variant="text"
            onClick={() => {}}
          >
            <Icon.Delete style={{ color: "var(--pluto-gray-l10)" }} />
          </Button.Icon>
        </>
      )}
    </List.Item>
  );
};

const metaDataItem = Component.renderProp(MetaDataListItem);

export interface MetaDataProps {
  rangeKey: ranger.Key;
}

export const MetaData = (props: MetaDataProps) => {
  const { data, useListItem } = Ranger.useListKV();
  const listProps = List.use({ data });
  return (
    <Align.Space y>
      <Text.Text level="h4" shade={11} weight={450}>
        Metadata
      </Text.Text>
      <List.List<string, kv.Pair> data={data} useItem={useListItem} {...listProps}>
        <List.Items>{metaDataItem}</List.Items>
      </List.List>
    </Align.Space>
  );
};
