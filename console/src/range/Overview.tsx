// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Overview.css";

import { ontology, ranger, Synnax as BaseSynnax, TimeRange } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Divider,
  Form,
  Header,
  Input,
  List,
  Observe,
  Ranger,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { change, deep, kv, observe, Primitive } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { width } from "node_modules/@synnaxlabs/x/dist/src/spatial/box/box";
import { FC, ReactElement, useCallback, useMemo } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { Link } from "@/link";

export const OVERVIEW_TYPE = "overview";

export const overviewLayout: Layout.State = {
  key: "overview",
  windowKey: "overview",
  type: "overview",
  name: "Overview",
  location: "mosaic",
};

const formSchema = z.object({
  name: z.string().min(1, "Name must not be empty"),
  timeRange: TimeRange.z,
});

export const Overview: Layout.Renderer = ({ layoutKey }): ReactElement => {
  const client = Synnax.use();

  const rng = useQuery({
    queryKey: ["range", layoutKey],
    queryFn: async () => {
      if (client == null) return;
      return await client.ranges.retrieve(layoutKey);
    },
  });
  if (rng.isPending || rng.isError) return <div>Loading...</div>;
  return <Internal layoutKey={layoutKey} rng={rng.data} />;
};

interface InternalProps {
  layoutKey: string;
  rng: ranger.Range;
}

const Internal = ({ rng }: InternalProps): ReactElement => {
  const methods = Form.use({
    values: rng.payload,
    schema: formSchema,
  });

  const handleLink = Link.useCopyToClipboard();
  const handleCopyLink = useCallback(() => {
    handleLink({
      name: rng.name,
      ontologyID: {
        key: rng.key,
        type: "range",
      },
    });
  }, [rng, handleLink]);

  return (
    <Align.Space
      direction="y"
      style={{ padding: "5rem", maxWidth: "1200px", margin: "0 auto" }}
    >
      <Form.Form {...methods}>
        <Align.Space direction="x" justify="spaceBetween">
          <Form.TextField
            path="name"
            showLabel={false}
            inputProps={{ variant: "natural", level: "h1", placeholder: "Name" }}
          />
          <Align.Space direction="x" style={{ height: "fit-content" }} size="small">
            <Align.Space direction="x">
              <Button.Icon variant="text">
                <Icon.Python style={{ color: "var(--pluto-gray-l7)" }} />
              </Button.Icon>
              <Button.Icon>
                <Icon.TypeScript style={{ color: "var(--pluto-gray-l7)" }} />
              </Button.Icon>
            </Align.Space>
            <Divider.Divider direction="y" />
            <Button.Icon variant="text" onClick={handleCopyLink}>
              <Icon.Link />
            </Button.Icon>
          </Align.Space>
        </Align.Space>
      </Form.Form>
      <Align.Space direction="x" size="large">
        <Form.Field<number> path="timeRange.start" label="From">
          {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
        </Form.Field>
        <Text.WithIcon level="h4" startIcon={<Icon.Arrow.Right />} />
        <Form.Field<number> path="timeRange.end" label="To">
          {(p) => <Input.DateTime level="h4" variant="natural" {...p} />}
        </Form.Field>
      </Align.Space>
      <Divider.Divider direction="x" />
      <MetaData rng={rng} />
      <Divider.Divider direction="x" />
      <SubRanges rng={rng} />
    </Align.Space>
  );
};

interface MetaDataProps {
  rng: ranger.Range;
}

interface KVPair {
  key: string;
  value: string;
}

const item = componentRenderProp(List.Column.Item) as List.VirtualCoreProps<
  string,
  KVPair
>["children"];

interface UseSyncedFormProps<Z extends z.ZodTypeAny, O = Z> {
  initialValues: z.output<Z>;
  key: Primitive[];
  queryFn: (client: BaseSynnax) => Promise<z.output<Z>>;
  openObservable?: (client: BaseSynnax) => Promise<observe.ObservableAsyncCloseable<O>>;
  transformObservable?: (value: O) => z.output<Z>;
  applyObservable?: (value: O, ctx: Form.ContextValue<Z>) => void;
  applyChanges?: (value: z.output<Z>, path: string) => Promise<void>;
}

interface MutationArgs<Z extends z.ZodTypeAny> {
  values: z.output<Z>;
  path: string;
}

export const useSyncedForm = <Z extends z.ZodTypeAny, O = Z>({
  key,
  queryFn,
  openObservable,
  transformObservable,
  applyChanges,
  initialValues: pInitialValues,
  applyObservable,
}: UseSyncedFormProps<Z, O>): Form.UseReturn<Z> => {
  const client = Synnax.use();
  const initialValues = useQuery({
    queryKey: [...key, "initialValues"],
    queryFn: async () => {
      if (client == null) return;
      return await queryFn(client);
    },
  });

  const methods = Form.use({
    values: initialValues.data ?? pInitialValues,
    sync: true,
    onChange: (values: z.output<Z>, path: string, prev: string) => {
      void applyChanges?.(values, path, prev);
    },
  });
  Observe.useListener({
    key,
    open: openObservable,
    onChange: (value) => {
      if (applyObservable != null) applyObservable(value, methods);
      else if (transformObservable != null) methods.set("", transformObservable(value));
      else methods.set("", value as z.output<Z>);
    },
  });
  return methods;
};

const metaDataFormSchema = z.object({
  pairs: kv.stringPairZ.array(),
});

type MetaDataValues = z.infer<typeof metaDataFormSchema>;

const MetaDataListItem: FC<List.ItemProps> = (props) => {
  const { index } = props;
  const ctx = Form.useContext();
  const arr = Form.fieldArrayUtils(ctx, "pairs");
  const key = ctx.get<string>(`pairs.${index}.key`, { optional: true })?.value;
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
          placeholder: "Key",
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
          <Form.TextField
            path={`pairs.${index}.value`}
            showLabel={false}
            showHelpText={false}
            hideIfNull
            inputProps={{
              style: { width: "unset", flexGrow: 2 },
              variant: "shadow",
              selectOnFocus: true,
              resetOnBlurIfEmpty: true,
              onlyChangeOnBlur: true,
              placeholder: "Value",
            }}
          />
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

const MetaData = ({ rng }: MetaDataProps) => {
  const formMethods = useSyncedForm<
    typeof metaDataFormSchema,
    change.Change<string, ranger.KVPair>[]
  >({
    initialValues: { pairs: [] },
    key: ["range", rng.key, "metadata"],
    queryFn: async () => {
      const res = await rng.kv.list();
      const pairs = Object.entries(res).map(([key, value]) => ({ key, value }));
      pairs.push({ key: "", value: "" });
      return { pairs };
    },
    openObservable: async () => await rng.kv.openTracker(),
    applyObservable: (changes, ctx) => {
      // const existingPairs = ctx.get<kv.Pair[]>("pairs").value;
      // const fu = Form.fieldArrayUtils(ctx, "pairs");
      // changes.map((c) => {
      //   const pos = existingPairs.findIndex((p) => p.key === c.value?.key);
      //   if (c.variant === "set") {
      //     if (pos === -1) fu.push({ key: c.value.key, value: c.value.value });
      //     if (existingPairs[pos].value == c.value.value) return;
      //     ctx.set(`pairs.${pos}.value`, c.value.value);
      //   } else if (c.variant === "delete" && pos !== -1) ctx.remove(`pairs.${pos}`);
      // });
    },
    applyChanges: async (
      value: MetaDataValues,
      path: string,
      prev: string | KVPair[],
    ) => {
      if (path === "pairs") {
        if (value.pairs.length >= prev.length) return;
        // a key was removed, take the difference and delete the key
        const newKeys = value.pairs.map((v) => v.key);
        const diff = (prev as KVPair[]).filter((p) => !newKeys.includes(p.key));
        console.log(diff);
        if (diff.length === 0) return;
        await rng.kv.delete(diff[0].key);
        return;
      }
      const split = path.split(".").slice(0, -1).join(".");
      const pair = deep.get<kv.Pair>(value, split, { optional: true });
      if (pair == null || pair.key === "") return;
      if (path.includes("key")) await rng.kv.delete(prev);
      await rng.kv.set(pair.key, pair.value);
    },
  });

  const arr = Form.useFieldArray<KVPair>({
    path: "pairs",
    ctx: formMethods,
    updateOnChildren: false,
  });

  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={8} weight={500}>
        Meta Data
      </Text.Text>
      <Form.Form {...formMethods}>
        <List.List<string, KVPair> data={arr.value}>
          <List.Core>{metaDataItem}</List.Core>
        </List.List>
      </Form.Form>
    </Align.Space>
  );
};

export const SubRangeListItem = (props: List.ItemProps<string, ranger.Payload>) => {
  const { entry } = props;
  return (
    <List.ItemFrame direction="y" size={0.5} style={{ padding: "1.5rem" }} {...props}>
      <Text.WithIcon
        startIcon={
          <Icon.Range
            style={{ transform: "scale(0.9)", color: "var(--pluto-gray-l9)" }}
          />
        }
        level="p"
        weight={450}
        shade={9}
        size="small"
      >
        {entry.name}{" "}
      </Text.WithIcon>
      <Ranger.TimeRangeChip level="small" timeRange={entry.timeRange} />
    </List.ItemFrame>
  );
};

const subRangeListItem = componentRenderProp(SubRangeListItem);

export const SubRanges: FC<{ rng: ranger.Range }> = ({ rng }) => {
  const client = Synnax.use();
  const subRanges = useQuery({
    queryKey: ["subranges", rng.key],
    queryFn: async () => {
      const res = await client?.ontology.retrieveChildren(
        new ontology.ID({
          key: rng.key.toString(),
          type: "range",
        }),
      );
      console.log(res);
      return res?.map((r) => ({
        key: r.data?.key,
        name: r.data?.name,
        timeRange: r.data?.timeRange,
      }));
    },
  });
  return (
    <Align.Space direction="y">
      <Text.Text level="h4" shade={8} weight={500}>
        Sub Ranges
      </Text.Text>
      <List.List data={subRanges.data ?? []}>
        <List.Core empty>{subRangeListItem}</List.Core>
      </List.List>
      <Button.Button
        size="small"
        shade={8}
        weight={500}
        startIcon={<Icon.Add />}
        variant="text"
        onClick={() => {}}
      >
        Add Sub Range
      </Button.Button>
    </Align.Space>
  );
};
