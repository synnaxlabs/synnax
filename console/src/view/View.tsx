// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/View.css";

import { type ontology, type view } from "@synnaxlabs/client";
import {
  Button,
  type Component,
  CSS as PCSS,
  Dialog,
  Flex,
  type Flux,
  Icon,
  Input,
  List,
  Select,
  type state,
  Status,
  useInactivity,
  View as PView,
} from "@synnaxlabs/pluto";
import { location, type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { useCallback, useState } from "react";

import { EmptyAction } from "@/components";
import { CSS } from "@/css";
import { Modals } from "@/modals";

export interface Request extends List.PagerParams, record.Unknown {}

export interface FiltersProps<R extends Request> {
  request: R;
  onRequestChange: state.Setter<R>;
}

export interface ViewProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  R extends Request,
> extends Pick<
    Flux.UseListReturn<R, K, E>,
    "data" | "getItem" | "subscribe" | "retrieve"
  > {
  onCreate: () => void;
  filters?: React.FC<FiltersProps<R>>;
  initialRequest: R;
  shownFilters?: React.FC<FiltersProps<R>>;
  resourceType: ontology.ResourceType;
  item: Component.RenderProp<List.ItemProps<K>>;
}

export const View = <
  K extends record.Key,
  E extends record.Keyed<K>,
  R extends Request,
>({
  data,
  getItem,
  retrieve,
  item,
  subscribe,
  initialRequest,
  filters,
  shownFilters,
  onCreate,
  resourceType,
}: ViewProps<K, E, R>) => {
  const [request, setRequest] = useState<R>(initialRequest);
  const [editable, setEditable] = useState(true);
  const [selected, setSelected] = useState<K[]>([]);
  const { visible, ref } = useInactivity<HTMLDivElement>(500);
  const { update: create } = PView.useCreate();
  const handleError = Status.useErrorHandler();
  const renameModal = Modals.useRename();
  const handleCreateView = useCallback(() => {
    handleError(async () => {
      const name = await renameModal(
        { initialValue: `View for ${resourceType}` },
        { icon: "Status", name: "View.Create" },
      );
      if (name == null) return;
      create({
        name,
        type: resourceType,
        query: request,
      });
    }, "Failed to create view");
  }, [create, request, resourceType, renameModal]);
  const handleRequestChange = useCallback(
    (setter: state.SetArg<R>, opts?: Flux.AsyncListOptions) => {
      if (typeof setter === "function")
        retrieve((p) => setter({ ...request, ...p }), opts);
      else retrieve(setter, opts);
      setRequest(setter);
    },
    [retrieve, request],
  );
  const handleSearch = useCallback(
    (searchTerm: string) => {
      handleRequestChange((p) => ({ ...p, ...List.search(p, searchTerm) }));
    },
    [handleRequestChange],
  );
  const handleEditableClick = useCallback(
    () => setEditable((editable) => !editable),
    [],
  );
  const handleFetchMore = useCallback(
    () =>
      handleRequestChange((p) => ({ ...p, ...List.page(p, 25) }), { mode: "append" }),
    [handleRequestChange],
  );
  const handleSelectView = useCallback((view: view.View) => {
    handleRequestChange(view.query as R);
  }, []);
  return (
    <Flex.Box full="y" empty className={CSS.B("view")} ref={ref}>
      <Select.Frame
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={handleFetchMore}
      >
        {editable && (
          <Flex.Box
            x
            bordered
            style={{ padding: "1.5rem" }}
            background={1}
            justify="between"
            align="center"
          >
            {filters != null && (
              <Filters request={request} onRequestChange={handleRequestChange}>
                {filters}
              </Filters>
            )}
            {shownFilters != null && (
              <>{shownFilters({ request, onRequestChange: handleRequestChange })}</>
            )}
            <SearchBox
              value={request.searchTerm ?? ""}
              resourceType={resourceType}
              onChange={handleSearch}
            />
          </Flex.Box>
        )}
        <Buttons
          onCreate={onCreate}
          resourceType={resourceType}
          onCreateView={handleCreateView}
          visible={visible}
          editable={editable}
          onEditableClick={handleEditableClick}
        />
        {editable && (
          <Views resourceType={resourceType} onSelectView={handleSelectView} />
        )}
        <List.Items<K>
          emptyContent={
            <EmptyContent onCreate={onCreate} resourceType={resourceType} />
          }
          grow
        >
          {item}
        </List.Items>
      </Select.Frame>
    </Flex.Box>
  );
};

interface ViewsProps {
  resourceType: string;
  onSelectView: (view: view.View) => void;
}

const Views = ({ resourceType, onSelectView }: ViewsProps) => {
  const listProps = PView.useList({ initialQuery: { types: [resourceType] } });
  if (listProps.data.length === 0) return null;
  return (
    <List.Frame<string, view.View> {...listProps}>
      <List.Items<string, view.View>
        displayItems={Infinity}
        x
        gap="medium"
        bordered
        align="center"
        style={{ padding: "1rem 1.5rem" }}
      >
        {({ key, ...rest }) => <Item key={key} {...rest} onSelectView={onSelectView} />}
      </List.Items>
    </List.Frame>
  );
};

interface ItemProps extends List.ItemProps<string> {
  onSelectView: (view: view.View) => void;
}

const Item = ({ itemKey, onSelectView }: ItemProps) => {
  const { getItem } = List.useUtilContext<string, view.View>();
  const { update: del } = PView.useDelete();
  const view = getItem?.(itemKey);
  if (view == null) return null;
  return (
    <Flex.Box x pack>
      <Button.Button onClick={() => onSelectView(view)}>{view.name}</Button.Button>
      <Button.Button onClick={() => del(itemKey)}>
        <Icon.Delete />
      </Button.Button>
    </Flex.Box>
  );
};

interface EmptyContentProps {
  onCreate: () => void;
  resourceType: string;
}

const EmptyContent = ({ onCreate, resourceType }: EmptyContentProps) => (
  <EmptyAction
    message={`No ${plural(resourceType)} created.`}
    action={`Create a ${resourceType}`}
    onClick={onCreate}
  />
);

interface FiltersContentProps<R extends Request> {
  request: R;
  onRequestChange: state.Setter<R>;
  children: React.FC<FiltersProps<R>>;
}

const Filters = <R extends Request>({
  request,
  onRequestChange,
  children,
}: FiltersContentProps<R>) => (
  <Dialog.Frame>
    <Dialog.Trigger hideCaret tooltip="Filter">
      <Icon.Filter />
    </Dialog.Trigger>
    <Dialog.Dialog
      background={1}
      bordered={false}
      pack={false}
      style={{ padding: "1rem" }}
    >
      <>{children({ request, onRequestChange })}</>
    </Dialog.Dialog>
  </Dialog.Frame>
);

interface SearchBoxProps extends Omit<Input.TextProps, "placeholder"> {
  resourceType: string;
}

const SearchBox = ({ resourceType, ...rest }: SearchBoxProps) => (
  <Input.Text
    size="small"
    level="h5"
    variant="text"
    {...rest}
    placeholder={`Search ${plural(resourceType)}...`}
  />
);

interface ButtonsProps {
  visible: boolean;
  hasSaveView?: boolean;
  onCreate: () => void;
  resourceType: string;
  editable: boolean;
  onEditableClick: () => void;
  onCreateView?: () => void;
}

const Buttons = ({
  visible,
  onCreate,
  onEditableClick,
  resourceType,
  editable,
  onCreateView,
}: ButtonsProps) => (
  <Flex.Box x className={CSS(CSS.BE("view", "buttons"), PCSS.visible(visible))} pack>
    <Button.Button
      onClick={onCreate}
      tooltipLocation={location.BOTTOM_LEFT}
      tooltip={`Create a ${resourceType}`}
    >
      <Icon.Add />
    </Button.Button>
    <Button.Toggle
      checkedVariant="filled"
      value={editable}
      onChange={onEditableClick}
      tooltipLocation={location.BOTTOM_LEFT}
      tooltip={`${editable ? "Disable" : "Enable"} editing`}
    >
      {editable ? <Icon.EditOff /> : <Icon.Edit />}
    </Button.Toggle>
    <Button.Button
      onClick={onCreateView}
      tooltipLocation={location.BOTTOM_LEFT}
      tooltip="Create a view"
    >
      <Icon.View />
    </Button.Button>
  </Flex.Box>
);
