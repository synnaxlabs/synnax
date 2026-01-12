// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import {
  type Flux,
  Form as PForm,
  List,
  Select,
  View as PView,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { FormContext, useContext } from "@/view/context";

export interface Query extends List.PagerParams, record.Unknown {}

export interface FormProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  Q extends Query,
>
  extends PropsWithChildren, Flux.UseListReturn<Q, K, E> {}

export const Form = <K extends record.Key, E extends record.Keyed<K>, Q extends Query>({
  children,
  data,
  getItem,
  retrieve,
  subscribe,
}: FormProps<K, E, Q>): ReactElement | null => {
  const { staticViews, getInitialView } = useContext("View.Form");
  const { fetchMore, search } = List.usePager({
    // type assertion here to deal with the weird setter<Q, Partial<Q>> type that causes
    // typing issues.
    retrieve: retrieve as List.UsePagerArgs["retrieve"],
    pageSize: 50,
  });
  const updateQuery = useCallback(
    (query: Q) => {
      retrieve((p) => ({ ...p, ...query, offset: 0, limit: 50, searchTerm: "" }));
    },
    [retrieve],
  );
  const { form } = PView.useForm({
    query: {},
    initialValues: getInitialView(),
    autoSave: true,
    beforeSave: useCallback(
      async ({
        value,
      }: Flux.FormBeforeSaveParams<
        PView.FormQuery,
        typeof view.newZ,
        PView.FluxSubStore
      >) => {
        const { key, query } = value();
        // if this is a static view we need to handle it here. Otherwise, the
        // useSetSynchronizer will handle it as it also needs to handle remote updates.
        if (key == null || !staticViews.includes(key)) return true;
        updateQuery(query as Q);
        return false;
      },
      [staticViews, updateQuery],
    ),
  });
  const handleSet = useCallback(
    (view: view.View) => {
      if (view.key !== form.get<string>("key").value) return;
      updateQuery(view.query as Q);
    },
    [form.get, updateQuery],
  );
  PView.useSetSynchronizer(handleSet);
  const [selected, setSelected] = useState<K[]>([]);
  const contextValue = useMemo(() => ({ search }), [search]);
  return (
    <PForm.Form<typeof view.newZ> {...form}>
      <Select.Frame
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={fetchMore}
      >
        <FormContext value={contextValue}>{children}</FormContext>
      </Select.Frame>
    </PForm.Form>
  );
};
