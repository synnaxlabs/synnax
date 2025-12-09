import { UnexpectedError, type view } from "@synnaxlabs/client";
import {
  type Flux,
  Form as PForm,
  List,
  Select,
  View as PView,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useMemo, useState } from "react";

import { FormContext, useContext } from "./context";

export interface Query extends List.PagerParams, record.Unknown {}

export interface FormProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  Q extends Query,
> extends PropsWithChildren,
    Flux.UseListReturn<Q, K, E> {}

export const Form = <K extends record.Key, E extends record.Keyed<K>, Q extends Query>(
  props: FormProps<K, E, Q>,
): ReactElement => {
  const { selected } = useContext("View.View");
  return <Internal key={selected} {...props} />;
};

const Internal = <K extends record.Key, E extends record.Keyed<K>, Q extends Query>({
  children,
  data,
  getItem,
  retrieve,
  subscribe,
}: FormProps<K, E, Q>): ReactElement => {
  const {
    getView,
    editable,
    selected: selectedView,
    staticViews,
  } = useContext("View.View");
  const { fetchMore, search } = List.usePager({
    // type assertion here to deal with the weird setter<Q, Partial<Q>> type that causes
    // typing issues.
    retrieve: retrieve as List.UsePagerArgs["retrieve"],
    pageSize: 50,
  });
  console.log("selectedView", selectedView);
  const initialView = getView(selectedView);
  console.log("initialView", initialView?.query);
  if (initialView == null) throw new UnexpectedError("No view found");
  const { form, save } = PView.useForm({
    query: {},
    initialValues: initialView,
    autoSave: true,
    beforeSave: async ({ value }) => {
      const { key, query } = value();
      const isStaticView = staticViews.has(key ?? "");
      retrieve((p) => {
        // type assertion because the current implementation of the query client doesn't
        // support custom typing yet.
        const queryObj = query as Q;
        // When switching to default view (All ResourceTypes) with empty query, start
        // with a clean base to ensure filter properties like hasLabels are cleared
        if (isStaticView && Object.keys(queryObj).length === 0) return query as Q;
        return { ...p, ...queryObj, offset: 0, limit: 100, searchTerm: "" };
      });
      return !isStaticView;
    },
  });
  const [selected, setSelected] = useState<K[]>([]);
  const contextValue = useMemo(
    () => ({ editable, search, save }),
    [editable, search, save],
  );
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
