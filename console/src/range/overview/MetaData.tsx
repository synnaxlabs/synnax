import { ranger } from "@synnaxlabs/client";
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
import { change, compare, deep, kv } from "@synnaxlabs/x";
import { FC, useMemo } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { useCopyToClipboard } from "@/hooks/useCopyToClipboard";

interface MetaDataProps {
  rangeKey: string;
}

const metaDataFormSchema = z.object({
  pairs: kv.stringPairZ.array(),
});

const MetaDataListItem: FC<List.ItemProps> = (props) => {
  const { index } = props;
  const ctx = Form.useContext();
  const arr = Form.fieldArrayUtils(ctx, "pairs");
  const key = ctx.get<string>(`pairs.${index}.key`, { optional: true })?.value;
  const copyToClipboard = useCopyToClipboard();
  return (
    <List.ItemFrame
      style={{ padding: "0.5rem", border: "none" }}
      className={CSS.BE("meta-data", "item")}
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
            {(p) => (
              <Input.Text
                {...p}
                style={{ width: "unset", flexGrow: 2 }}
                variant="shadow"
                selectOnFocus={true}
                resetOnBlurIfEmpty={true}
                onlyChangeOnBlur={true}
                placeholder="Value"
              >
                <Button.Icon
                  onClick={() => {
                    copyToClipboard(p.value, "value");
                  }}
                >
                  <Icon.Copy />
                </Button.Icon>
              </Input.Text>
            )}
          </Form.Field>
          <Button.Icon
            className={CSS.BE("meta-data", "delete")}
            size="small"
            variant="text"
            onClick={() => {}}
          >
            <Icon.Delete
              style={{ color: "var(--pluto-gray-l8)" }}
              onClick={() => arr.remove(index)}
            />
          </Button.Icon>
        </>
      )}
    </List.ItemFrame>
  );
};

const metaDataItem = componentRenderProp(MetaDataListItem);

export const MetaData = ({ rangeKey }: MetaDataProps) => {
  const formCtx = Form.useSynced<
    typeof metaDataFormSchema,
    change.Change<string, ranger.KVPair>[]
  >({
    values: { pairs: [] },
    name: "Range Meta Data",
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
      const fu = Form.fieldArrayUtils(ctx, "pairs");
      changes.map((c) => {
        const pos = existingPairs.findIndex((p) => p.key === c.value?.key);
        if (c.variant === "set") {
          if (pos === -1) fu.push({ key: c.value.key, value: c.value.value });
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
      const pair = deep.get<kv.Pair>(values, split, { optional: true });
      if (pair == null || pair.key === "") return;
      if (path.includes("key")) await kv.delete(prev as string);
      await kv.set(pair.key, pair.value);
    },
  });
  const arr = Form.useFieldArray<kv.Pair>({ path: "pairs", ctx: formCtx });
  const sorted = useMemo(
    () =>
      arr.value.sort((a, b) => {
        if (a.key === "") return 1;
        if (b.key === "") return -1;
        return compare.stringsWithNumbers(a.key, b.key);
      }),
    [arr.value],
  );
  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={8} weight={500}>
        Meta Data
      </Text.Text>
      <Form.Form {...formCtx}>
        <List.List<string, kv.Pair> data={sorted}>
          <List.Core>{metaDataItem}</List.Core>
        </List.List>
      </Form.Form>
    </Align.Space>
  );
};
