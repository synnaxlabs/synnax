// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Divider,
  Form,
  Input,
  List,
  Text,
} from "@synnaxlabs/pluto";
import { type change, compare, deep, kv, link } from "@synnaxlabs/x";
import { type FC, type ReactElement, useMemo } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

interface MetaDataProps {
  rangeKey: string;
}

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const metaDataFormSchema = z.object({
  pairs: kv.stringPairZ.array(),
});

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
      color={isLink ? "var(--pluto-primary-z)" : "var(--pluto-gray-l8)"}
    >
      <Button.Icon onClick={() => copyToClipboard(value, "value")}>
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

const valueInput = componentRenderProp(ValueInput);

const MetaDataListItem: FC<List.ItemProps> = (props) => {
  const { index } = props;
  const ctx = Form.useContext();
  const arr = Form.fieldArrayUtils(ctx, "pairs");
  const key = ctx.get<string>(`pairs.${index}.key`, { optional: true })?.value;
  return (
    <List.ItemFrame
      style={{ padding: "0.5rem", border: "none" }}
      className={CSS.BE("metadata", "item")}
      allowSelect={false}
      {...props}
    >
      <Form.TextField
        path={`pairs.${index}.key`}
        showLabel={false}
        showHelpText={false}
        hideIfNull
        inputProps={{
          style: {
            flexBasis: "30%",
            width: 250,
          },
          variant: "shadow",
          selectOnFocus: true,
          resetOnBlurIfEmpty: true,
          onlyChangeOnBlur: true,
          placeholder: "Add Key",
          weight: 500,
        }}
        onChange={(value, ctx) => {
          const v = ctx.get<string>(`pairs.${index}.value`).value;
          const pairsLength = ctx.get<kv.Pair[]>("pairs").value.length;
          if (v.length === 0 && value.length > 0 && index === pairsLength - 1)
            arr.push({ key: "", value: "" });
        }}
      />
      <Divider.Divider direction="y" />

      {key != null && key.length !== 0 && (
        <>
          <Form.Field<string>
            path={`pairs.${index}.value`}
            showLabel={false}
            showHelpText={false}
            hideIfNull
          >
            {valueInput}
          </Form.Field>
          <Button.Icon
            className={CSS.BE("metadata", "delete")}
            size="small"
            variant="text"
            onClick={() => {
              arr.remove(index);
            }}
          >
            <Icon.Delete style={{ color: "var(--pluto-gray-l8)" }} />
          </Button.Icon>
        </>
      )}
    </List.ItemFrame>
  );
};

const metaDataItem = componentRenderProp(MetaDataListItem);

const sortF: compare.CompareF<kv.Pair> = (a, b) => {
  if (a.key === "") return 1;
  if (b.key === "") return -1;
  return compare.stringsWithNumbers(a.key, b.key);
};

export const MetaData = ({ rangeKey }: MetaDataProps) => {
  const formCtx = Form.useSynced<
    typeof metaDataFormSchema,
    change.Change<string, ranger.KVPair>[]
  >({
    values: { pairs: [] },
    name: "Range Metadata",
    key: ["range", rangeKey, "metadata"],
    queryFn: async ({ client }) => {
      const kv = client.ranges.getKV(rangeKey);
      const res = await kv.list();
      const pairs = Object.entries(res).map(([key, value]) => ({ key, value }));
      pairs.push({ key: "", value: "" });
      return { pairs };
    },
    openObservable: async (client) => await client.ranges.getKV(rangeKey).openTracker(),
    applyObservable: ({ changes, ctx }) => {
      const existingPairs = ctx.get<kv.Pair[]>("pairs").value;
      const fu = Form.fieldArrayUtils<kv.Pair>(ctx, "pairs");
      changes
        .filter((c) => c.value?.range === rangeKey)
        .map((c) => {
          const pos = existingPairs.findIndex((p) => p.key === c.value?.key);
          if (c.variant === "set") {
            if (pos === -1)
              return fu.push({ key: c.value.key, value: c.value.value }, sortF);
            if (existingPairs[pos].value == c.value.value) return;
            ctx.set(`pairs.${pos}.value`, c.value.value);
          } else if (c.variant === "delete" && pos !== -1) ctx.remove(`pairs.${pos}`);
        });
    },
    applyChanges: async ({ client, values, path, prev }) => {
      if (path === "") return;
      const kv = client.ranges.getKV(rangeKey);
      if (path === "pairs") {
        const tPrev = prev as kv.Pair[];
        if (values.pairs.length >= tPrev.length) return;
        // a key was removed, take the difference and delete the key
        const newKeys = values.pairs.map((v) => v.key);
        const diff = tPrev.filter((p) => !newKeys.includes(p.key));
        if (diff.length === 0) return;
        await kv.delete(diff[0].key);
        return;
      }
      const split = path.split(".").slice(0, -1).join(".");
      const pair = deep.get<kv.Pair<string>>(values, split, { optional: true });
      if (pair == null || pair.key === "") return;
      if (path.includes("key")) await kv.delete(prev as string);
      await kv.set(pair.key, pair.value);
    },
  });
  const arr = Form.useFieldArray<kv.Pair>({ path: "pairs", ctx: formCtx });
  const sorted = useMemo(() => arr.value.sort(), [arr.value]);
  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={9} weight={450}>
        Metadata
      </Text.Text>
      <Form.Form {...formCtx}>
        <List.List<string, kv.Pair> data={sorted}>
          <List.Core>{metaDataItem}</List.Core>
        </List.List>
      </Form.Form>
    </Align.Space>
  );
};
