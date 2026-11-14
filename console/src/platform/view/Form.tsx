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
import { Select } from "@synnaxlabs/lyra/select";
import { type Flux, View as PView } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";

import { FormContext, useContext } from "@/platform/view/context";

const path = PView.formPath;

export interface FormProps<K extends record.Key, E extends record.Keyed<K>>
  extends
    PropsWithChildren,
    Pick<
      Flux.UseListReturn<K, E>,
      "data" | "getItem" | "subscribe" | "answered" | "more" | "search"
    > {
  /** Called with the filter of the view each time the selected view changes it. */
  onWhereChange: (where: view.View["where"]) => void;
  /** The height of each item in pixels. The list renders only the items in view. */
  itemHeight: number;
}

export const Form = <K extends record.Key, E extends record.Keyed<K>>({
  children,
  data,
  getItem,
  subscribe,
  answered,
  more,
  search,
  onWhereChange,
  itemHeight,
}: FormProps<K, E>): ReactElement | null => {
  const { staticViews, getInitialView, searchTerm } = useContext("View.Form");
  // The list of a new view starts with no term.
  const searchedRef = useRef("");
  useEffect(() => {
    if (searchTerm === searchedRef.current) return;
    searchedRef.current = searchTerm;
    search(searchTerm);
  }, [searchTerm]);
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
      onWhereChange(where);
      return false;
    },
  });
  const { get: getField } = form;
  const handleSet = (view: view.View) => {
    if (view.key !== getField(path.key).value) return;
    onWhereChange(view.where);
  };
  PView.useSetSynchronizer(handleSet);
  const [selected, setSelected] = useState<K[]>([]);
  const contextValue = { answered };
  return (
    <PForm.Form<typeof PView.formSchema> {...form}>
      <Select.Frame
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={more}
        virtual
        itemHeight={itemHeight}
      >
        <FormContext value={contextValue}>{children}</FormContext>
      </Select.Frame>
    </PForm.Form>
  );
};
