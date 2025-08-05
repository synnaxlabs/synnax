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
  Button,
  Component,
  Divider,
  Flex,
  Header,
  Icon,
  Input,
  List,
  Ranger,
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
      selectOnFocus={true}
      resetOnBlurIfEmpty={true}
      onlyChangeOnBlur={true}
      placeholder="Value"
      color={isLink ? "var(--pluto-primary-z)" : "var(--pluto-gray-l10)"}
    >
      <Button.Button onClick={() => copyToClipboard(value, "value")} variant="outlined">
        <Icon.Copy />
      </Button.Button>
      {isLink && (
        <Button.Button
          variant="outlined"
          href={value}
          target="_blank"
          autoFormatHref
          style={{ padding: "1rem" }}
        >
          <Icon.LinkExternal />
        </Button.Button>
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
          <Button.Button
            className={CSS.BE("metadata", "delete")}
            size="small"
            variant="text"
            onClick={() => {}}
          >
            <Icon.Delete style={{ color: "var(--pluto-gray-l10)" }} />
          </Button.Button>
        </>
      )}
    </List.Item>
  );
};

const metaDataItem = Component.renderProp(MetaDataListItem);

export interface MetaDataProps {
  rangeKey: ranger.Key;
}

export const MetaData = ({ rangeKey }: MetaDataProps): ReactElement => {
  const { data, getItem, subscribe } = Ranger.useListKV({
    initialParams: { rangeKey },
  });
  return (
    <Flex.Box y>
      <Header.Header level="h4" bordered={false} borderColor={10}>
        <Header.Title color={10} weight={450}>
          Metadata
        </Header.Title>
      </Header.Header>
      <List.Frame<string, kv.Pair> data={data} getItem={getItem} subscribe={subscribe}>
        <List.Items>{metaDataItem}</List.Items>
      </List.Frame>
    </Flex.Box>
  );
};
