// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/Overview.css";

import { ranger, Synnax as BaseSynnax, TimeRange } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  componentRenderProp,
  Form,
  Input,
  List,
  Observe,
  Synnax,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { observe, Primitive } from "@synnaxlabs/x";
import { useQuery } from "@tanstack/react-query";
import { ReactElement, useCallback, useMemo } from "react";
import { z } from "zod";

import { Layout } from "@/layout";

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

const Internal: Layout.Renderer = ({ rng }: InternalProps): ReactElement => {
  const methods = Form.use({
    values: rng.payload,
    schema: formSchema,
  });

  return (
    <Align.Space direction="y" style={{ padding: "5rem" }}>
      <Form.Form {...methods}>
        <Form.TextField
          path="name"
          showLabel={false}
          inputProps={{ variant: "natural", level: "h1", placeholder: "Name" }}
        />
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

const KV_COLUMNS: List.ColumnSpec<string, KVPair>[] = [];

const item = componentRenderProp(List.Column.Item) as List.VirtualCoreProps<
  string,
  KVPair
>["children"];

interface UseSyncedFormProps<Z extends z.ZodTypeAny, O = Z> {
  key: Primitive[];
  queryFn: (client: BaseSynnax) => Promise<z.output<Z>>;
  openObservable?: (client: BaseSynnax) => Promise<observe.ObservableAsyncCloseable<O>>;
  transformObservable?: (value: O) => z.output<Z>;
  applyObservable?: (value: O, ctx: Form.ContextValue<Z>) => void;
}

export const useSyncedForm = <Z extends z.ZodTypeAny, O = Z>({
  key,
  queryFn,
  openObservable,
  transformObservable,
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
    values: initialValues.data,
    sync: false,
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

const MetaData = ({ rng }: MetaDataProps) => {
  const client = Synnax.use();

  const kv = useSyncedForm({
    key: ["range", rng.key, "metadata"],
    openObservable: async () => {
      if (client == null) return;
      return await rng.kv.
    }
  });
  

  const columns = useMemo(() => {
    const columns: List.ColumnSpec<string, KVPair>[] = [
      {
        key: "key",
        name: "Value",
        render: ({ entry }) => (
          <Input.Text value={entry.key} onChange={handleKeyChange} />
        ),
      },
      {
        key: "key",
        name: "Value",
        render: ({ entry }) => <Input.Text value={entry.value} />,
      },
    ];
  }, [rng]);

  if (kv.isPending || kv.isError) return <div>Loading...</div>;

  return (
    <List.List<string, KVPair> data={kv}>
      <List.Column.Header hide={true} columns={KV_COLUMNS}>
        <List.Core>{item}</List.Core>
      </List.Column.Header>
    </List.List>
  );
};
