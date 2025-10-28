// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/View.css";

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
} from "@synnaxlabs/pluto";
import { location, type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { useCallback, useEffect, useRef, useState } from "react";

import { EmptyAction } from "@/components";
import { CSS } from "@/css";

interface SearchBoxProps {
  searchTerm: string;
  resourceType: string;
  onChange: (searchTerm: string) => void;
}

const SearchBox = ({ searchTerm, resourceType, onChange }: SearchBoxProps) => (
  <Input.Text
    size="small"
    level="h5"
    variant="text"
    placeholder={`Search ${plural(resourceType)}...`}
    value={searchTerm}
    onChange={onChange}
  />
);
export interface FiltersProps<R extends Request> {
  request: R;
  onRequestChange: state.Setter<R, Partial<R>>;
}

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
  hasSaveView,
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
    {hasSaveView && (
      <Button.Button
        onClick={onCreateView}
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip="Save as view"
      >
        <Icon.View />
      </Button.Button>
    )}
  </Flex.Box>
);

export interface Request extends List.PagerParams, record.Unknown {}

export interface ViewProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  R extends Request,
> extends Pick<
    Flux.UseListReturn<R, K, E>,
    "data" | "getItem" | "subscribe" | "retrieve"
  > {
  request: R;
  onRequestChange: state.Setter<R>;
  onCreate: () => void;
  filters?: React.FC<FiltersProps<R>>;
  shownFilters?: React.FC<FiltersProps<R>>;
  onCreateView?: () => void;
  resourceType: string;
  views?: React.ReactElement;
  initialEditable: boolean;
  hasSaveView?: boolean;
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
  views,
  filters,
  shownFilters,
  request,
  onRequestChange,
  onCreate,
  onCreateView,
  resourceType,
  initialEditable,
  hasSaveView = false,
}: ViewProps<K, E, R>) => {
  const [editable, setEditable] = useState(initialEditable);
  const { fetchMore, search } = List.usePager({ retrieve });
  const [selected, setSelected] = useState<K[]>([]);
  const [showControls, setShowControls] = useState(false);
  const inactivityTimeoutRef = useRef<NodeJS.Timeout>(undefined);
  const viewRef = useRef<HTMLDivElement>(null);

  const handleMouseMove = useCallback(() => {
    setShowControls(true);
    clearTimeout(inactivityTimeoutRef.current);
    inactivityTimeoutRef.current = setTimeout(() => {
      setShowControls(false);
    }, 500);
  }, []);
  const handleMouseLeave = useCallback(() => {
    setShowControls(false);
    clearTimeout(inactivityTimeoutRef.current);
  }, []);

  useEffect(() => {
    const el = viewRef.current;
    if (el == null) return;
    el.addEventListener("mousemove", handleMouseMove);
    el.addEventListener("mouseleave", handleMouseLeave);
    return () => {
      el.removeEventListener("mousemove", handleMouseMove);
      el.removeEventListener("mouseleave", handleMouseLeave);
      clearTimeout(inactivityTimeoutRef.current);
    };
  }, [handleMouseMove]);

  const handleRequestChange = useCallback(
    (setter: state.SetArg<R, Partial<R>>, opts?: Flux.AsyncListOptions) => {
      retrieve(setter, opts);
      onRequestChange(setter);
    },
    [retrieve, onRequestChange],
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
  return (
    <Flex.Box full="y" empty className={CSS.B("view")} ref={viewRef}>
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
              searchTerm={request.searchTerm ?? ""}
              resourceType={resourceType}
              onChange={handleSearch}
            />
          </Flex.Box>
        )}
        <Buttons
          hasSaveView={hasSaveView}
          onCreate={onCreate}
          resourceType={resourceType}
          onCreateView={onCreateView}
          visible={showControls}
          editable={editable}
          onEditableClick={handleEditableClick}
        />
        {editable && views != null && views}
        <List.Items<K>
          emptyContent={
            <EmptyContent onCreate={onCreate} resourceType={resourceType} />
          }
          displayItems={Infinity}
          grow
        >
          {item}
        </List.Items>
      </Select.Frame>
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
  onRequestChange: state.Setter<R, Partial<R>>;
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
