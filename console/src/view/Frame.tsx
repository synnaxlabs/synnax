// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/View.css";

import { type ontology, view } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Flex,
  type Flux,
  Form,
  Icon,
  List,
  Select,
  Status,
  View as PView,
} from "@synnaxlabs/pluto";
import { location, type record, uuid } from "@synnaxlabs/x";
import { plural } from "pluralize";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { Controls } from "@/components";
import { Modals } from "@/modals";
import { Provider } from "@/view/context";

export interface Query extends List.PagerParams, record.Unknown {}

export interface FrameProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  Q extends Query,
> extends PropsWithChildren,
    Pick<Flux.UseListReturn<Q, K, E>, "data" | "getItem" | "subscribe" | "retrieve"> {
  resourceType: ontology.ResourceType;
}

export const Frame = <
  K extends record.Key,
  E extends record.Keyed<K>,
  Q extends Query,
>({
  children,
  resourceType,
  data,
  getItem,
  retrieve,
  subscribe,
}: FrameProps<K, E, Q>): ReactElement => {
  const [selected, setSelected] = useState<K[]>([]);
  const { fetchMore, search } = List.usePager({
    // type assertion here to deal with the weird setter<Q, Partial<Q>> type that causes
    // typing issues.
    retrieve: retrieve as List.UsePagerArgs["retrieve"],
  });
  const defaultViewKey = useMemo(() => uuid.create(), []);
  const { form, save } = PView.useForm({
    query: {},
    initialValues: { type: resourceType, name: "", key: defaultViewKey, query: {} },
    autoSave: true,
    beforeSave: async ({ value }) => {
      const { key, query } = value();
      // type assertion because the current implementation of the query client doesn't
      // support custom typing yet.
      retrieve((p) => ({ ...p, ...(query as Q), offset: 0, limit: 25 }));
      return key !== defaultViewKey;
    },
  });
  const formKey = Form.useFieldValue<view.Key, view.Key, typeof view.newZ>("key", {
    ctx: form,
  });
  const isDefault = formKey === defaultViewKey;
  const canEditView = Access.useUpdateGranted(view.ontologyID(formKey ?? ""));
  const [editable, setEditable] = useState(canEditView);
  const contextValue = useMemo(
    () => ({ editable, resourceType, search, save, isDefault, defaultViewKey }),
    [editable, resourceType, search, save, isDefault, defaultViewKey],
  );
  const handleError = Status.useErrorHandler();
  const renameModal = Modals.useRename();
  const canCreate = Access.useCreateGranted(view.TYPE_ONTOLOGY_ID) && editable;
  const handleCreate = useCallback(() => {
    handleError(async () => {
      const name = await renameModal(
        { initialValue: `View for ${plural(resourceType)}` },
        { name: "View.Create" },
      );
      if (name == null) return;
      form.set("key", uuid.create());
      form.set("name", name);
    }, "Failed to create view");
  }, [renameModal, resourceType, form.set]);
  return (
    <Form.Form<typeof view.newZ> {...form}>
      <Flex.Box full="y" empty>
        <Controls x>
          {canCreate && (
            <Button.Button
              onClick={handleCreate}
              tooltip="Create a view"
              size="small"
              tooltipLocation={location.BOTTOM_LEFT}
            >
              <Icon.Add />
            </Button.Button>
          )}
          {canEditView && (
            <Button.Toggle
              size="small"
              value={editable}
              onChange={() => setEditable((prev) => !prev)}
              tooltipLocation={location.BOTTOM_LEFT}
              tooltip={`${editable ? "Disable" : "Enable"} editing`}
            >
              {editable ? <Icon.EditOff /> : <Icon.Edit />}
            </Button.Toggle>
          )}
        </Controls>
        <Select.Frame
          multiple
          data={data}
          getItem={getItem}
          subscribe={subscribe}
          onChange={setSelected}
          value={selected}
          onFetchMore={fetchMore}
        >
          <Provider value={contextValue}>{children}</Provider>
        </Select.Frame>
      </Flex.Box>
    </Form.Form>
  );
};
