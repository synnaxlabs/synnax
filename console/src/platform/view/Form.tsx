// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { type Flux, View as PView } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useState } from "react";

import { FormContext, useContext } from "@/platform/view/context";

export type Query = List.PagerParams;

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
  answered,
}: FormProps<K, E, Q>): ReactElement | null => {
  const { staticViews, getInitialView } = useContext("View.Form");
  // type assertion here to deal with the weird setter<Q, Partial<Q>> type that causes
  // typing issues.
  const pagerRetrieve = retrieve as List.UsePagerParams["retrieve"];
  const { fetchMore, search } = List.usePager({
    retrieve: pagerRetrieve,
    pageSize: 50,
  });
  const updateWhere = (where: view.View["where"]) => {
    pagerRetrieve((p) => ({ ...p, where, offset: 0, limit: 50, searchTerm: "" }));
  };
  const { form } = PView.useForm({
    query: null,
    initialValues: getInitialView(),
    autoSave: true,
    beforeSave: async ({
      value,
    }: Flux.FormBeforeSaveParams<PView.FormQuery, typeof PView.formSchema>) => {
      const { key, where } = value();
      // if this is a static view we need to handle it here. Otherwise, the
      // useSetSynchronizer will handle it as it also needs to handle remote updates.
      if (key == null || !staticViews.includes(key)) return true;
      updateWhere(where);
      return false;
    },
  });
  const { get: getField } = form;
  const handleSet = (view: view.View) => {
    if (view.key !== getField<string>("key").value) return;
    updateWhere(view.where);
  };
  PView.useSetSynchronizer(handleSet);
  const [selected, setSelected] = useState<K[]>([]);
  const contextValue = { search, answered };
  return (
    <PForm.Form<typeof PView.formSchema> {...form}>
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
