// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, ontology } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Component,
  Flex,
  Flux,
  Icon,
  List,
  Nav,
  Ontology,
  Status,
  Synnax,
  Text,
  Tree,
  useAsyncEffect,
  useCombinedStateAndRef,
} from "@synnaxlabs/pluto";
import {
  type ReactElement,
  useCallback,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";

import { Cluster } from "@/cluster";
import { downloadSyc,type ExportSycRequest } from "@/export/download";
import { type CheckedState, useCheckedState } from "@/export/useCheckedState";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { type Service } from "@/ontology/service";
import { useServices } from "@/ontology/ServicesProvider";
import { Triggers } from "@/triggers";

export const EXPORT_LAYOUT_TYPE = "exportSynnax";

export const EXPORT_LAYOUT: Layout.BaseState = {
  key: EXPORT_LAYOUT_TYPE,
  type: EXPORT_LAYOUT_TYPE,
  name: "Export Synnax",
  icon: "Export",
  location: "modal",
  window: { resizable: true, size: { height: 600, width: 500 }, navTop: true },
};

interface SectionConfig {
  key: string;
  label: string;
  icon: ReactElement;
}

const SECTIONS: SectionConfig[] = [
  { key: "section:workspaces", label: "Workspaces", icon: <Icon.Workspace /> },
  { key: "section:users", label: "Users", icon: <Icon.User /> },
  { key: "section:devices", label: "Devices", icon: <Icon.Device /> },
  { key: "section:tasks", label: "Tasks", icon: <Icon.Task /> },
  { key: "section:ranges", label: "Ranges", icon: <Icon.Range /> },
  { key: "section:channels", label: "Channels", icon: <Icon.Channel /> },
];

const SECTION_MAP = new Map(SECTIONS.map((s) => [s.key, s]));

const isSection = (key: string): boolean => key.startsWith("section:");

const INITIAL_NODES: Tree.Node[] = SECTIONS.map((s) => ({
  key: s.key,
  children: [],
}));

const SECTION_PREFIX = "section:";
const CHANNELS_SECTION_KEY = `${SECTION_PREFIX}channels`;

const resolveResourceIcon = (
  id: ontology.ID,
  resource: ontology.Resource,
  service: Service,
): ReactElement | undefined => {
  if (id.type === "channel") {
    const DataTypeIcon = Channel.resolveIcon(resource.data as channel.Payload);
    return <DataTypeIcon />;
  }
  return Icon.resolve(
    typeof service.icon === "function" ? service.icon(resource) : service.icon,
  );
};

interface IndeterminateCheckboxProps {
  checked: boolean;
  indeterminate: boolean;
  onChange: () => void;
}

const IndeterminateCheckbox = ({
  checked,
  indeterminate,
  onChange,
}: IndeterminateCheckboxProps): ReactElement => {
  const ref = useCallback(
    (el: HTMLInputElement | null) => {
      if (el != null) el.indeterminate = indeterminate;
    },
    [indeterminate],
  );
  return (
    <input
      type="checkbox"
      ref={ref}
      checked={checked}
      onChange={onChange}
      onClick={(e) => e.stopPropagation()}
      style={{ marginRight: "0.25rem" }}
    />
  );
};

interface CheckboxItemProps extends Tree.ItemProps<string> {
  nodes: Tree.Node[];
  checkedState: CheckedState;
  services: ReturnType<typeof useServices>;
  loadingRef: React.RefObject<string | false>;
  loadingListeners: React.RefObject<Set<() => void>>;
  channelCount: number;
  onCheck: (key: string) => void;
}

const CheckboxItem = ({
  nodes,
  checkedState,
  services,
  loadingRef,
  loadingListeners,
  channelCount,
  onCheck,
  ...props
}: CheckboxItemProps): ReactElement | null => {
  const { itemKey } = props;
  const { isChecked, isIndeterminate, checked } = checkedState;
  const itemChecked = isChecked(itemKey);
  const indeterminate = isIndeterminate(itemKey, nodes);
  const handleToggle = useCallback(
    () => onCheck(itemKey),
    [onCheck, itemKey],
  );

  const loading = useSyncExternalStore<boolean>(
    useCallback(
      (cb) => {
        loadingListeners.current.add(cb);
        return () => loadingListeners.current.delete(cb);
      },
      [loadingListeners],
    ),
    useCallback(() => loadingRef.current === itemKey, [itemKey, loadingRef]),
  );

  const section = SECTION_MAP.get(itemKey);
  if (section != null) {
    const isChannels = itemKey === CHANNELS_SECTION_KEY;
    let selectedCount = 0;
    if (isChannels && channelCount > 0) {
      const sectionNode = Tree.findNode({ tree: nodes, key: itemKey });
      const descendants =
        sectionNode != null ? Tree.getDescendants(sectionNode) : [];
      selectedCount = descendants.filter(
        (n) => n.key !== itemKey && !n.key.startsWith("group:") && checked.has(n.key),
      ).length;
    }
    return (
      <Tree.Item {...props} loading={loading}>
        <IndeterminateCheckbox
          checked={itemChecked}
          indeterminate={indeterminate}
          onChange={handleToggle}
        />
        {section.icon}
        <Text.Text level="small" weight={500} style={{ userSelect: "none" }}>
          {section.label}
        </Text.Text>
        {isChannels && channelCount > 0 && (
          <Text.Text
            level="small"
            style={{
              userSelect: "none",
              marginLeft: "0.5rem",
              color: "var(--pluto-gray-l5)",
            }}
          >
            ({selectedCount}/{channelCount})
          </Text.Text>
        )}
      </Tree.Item>
    );
  }

  const id = ontology.idZ.parse(itemKey);
  const resource = List.useItem<string, ontology.Resource>(itemKey);
  if (resource == null) return null;

  return (
    <Tree.Item {...props} loading={loading}>
      <IndeterminateCheckbox
        checked={itemChecked}
        indeterminate={indeterminate}
        onChange={handleToggle}
      />
      {resolveResourceIcon(id, resource, services[id.type])}
      <Text.Text
        level="small"
        style={{ userSelect: "none", flexGrow: 1, width: 0 }}
        overflow="ellipsis"
      >
        {resource.name}
      </Text.Text>
    </Tree.Item>
  );
};

const buildExportRequest = (checked: Set<string>): ExportSycRequest => {
  const request: ExportSycRequest = {};
  for (const key of checked) {
    if (key.startsWith("section:") || key.startsWith("group:")) continue;
    const id = ontology.idZ.safeParse(key);
    if (!id.success) continue;
    const { type, key: resourceKey } = id.data;
    switch (type) {
      case "workspace":
        (request.workspace_keys ??= []).push(resourceKey);
        break;
    }
  }
  return request;
};

export const ExportModal = (_: Layout.RendererProps): ReactElement => {
  const services = useServices();
  const [nodes, setNodes] = useCombinedStateAndRef<Tree.Node[]>(
    () => Tree.deepCopy(INITIAL_NODES),
  );
  const resourceStore = Flux.useStore<Ontology.FluxSubStore>().resources;
  const client = Synnax.use();
  const cluster = Cluster.useSelect();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const checkedState = useCheckedState();

  const loadingRef = useRef<string | false>(false);
  const loadingListenersRef = useRef(new Set<() => void>());
  const groupMapRef = useRef(new Map<string, ontology.ID>());
  const [channelCount, setChannelCount] = useState(0);

  const setLoading = useCallback((key: string | false) => {
    loadingRef.current = key;
    loadingListenersRef.current.forEach((cb) => cb());
  }, []);

  useAsyncEffect(async () => {
    if (client == null) return;
    try {
      const [rootChildren, channels] = await Promise.all([
        client.ontology.retrieveChildren(ontology.ROOT_ID),
        client.channels.retrieve({ internal: false }),
      ]);
      rootChildren.forEach((r) => resourceStore.set(r));
      for (const section of SECTIONS) {
        const group = rootChildren.find((r) => r.name === section.label);
        if (group != null) groupMapRef.current.set(section.key, group.id);
      }
      setChannelCount(channels.length);
    } catch (e) {
      handleError(e);
    }
  }, [client]);

  const retrieveChildren = Ontology.useRetrieveObservableChildren({
    onChange: useCallback(
      ({ data: resources, variant }, { id }) => {
        if (variant === "success") {
          const filtered = resources.filter((r) => {
            const svc = services[r.id.type];
            return svc.visible == null || svc.visible(r);
          });
          const converted = filtered.map((r) => ({
            key: ontology.idToString(r.id),
            children: services[r.id.type].hasChildren ? [] : undefined,
          }));
          const ids = new Set(filtered.map((r) => ontology.idToString(r.id)));
          const parentStr = ontology.idToString(id);
          let treeParent = parentStr;
          for (const [sectionKey, groupID] of groupMapRef.current.entries())
            if (ontology.idToString(groupID) === parentStr) {
              treeParent = sectionKey;
              break;
            }

          setNodes((prev) => {
            const next = [
              ...Tree.updateNodeChildren({
                tree: prev,
                parent: treeParent,
                updater: (prevChildren) => [
                  ...prevChildren.filter(({ key }) => !ids.has(key)),
                  ...converted,
                ],
              }),
            ];
            checkedState.reconcile(next);
            return next;
          });
        }
        setLoading(false);
      },
      [services, setLoading],
    ),
  });

  const handleExpand = useCallback(
    ({ action, clicked }: Tree.HandleExpandProps) => {
      if (action !== "expand") return;
      if (isSection(clicked)) {
        const groupID = groupMapRef.current.get(clicked);
        if (groupID != null) {
          setLoading(clicked);
          retrieveChildren.retrieve({ id: groupID });
        }
        return;
      }
      const clickedID = ontology.idZ.parse(clicked);
      setLoading(clicked);
      retrieveChildren.retrieve({ id: clickedID });
    },
    [retrieveChildren, setLoading],
  );

  const fetchNodeChildren = useCallback(
    (key: string) => {
      if (isSection(key)) {
        const groupID = groupMapRef.current.get(key);
        if (groupID != null) retrieveChildren.retrieve({ id: groupID });
      } else {
        const id = ontology.idZ.parse(key);
        retrieveChildren.retrieve({ id });
      }
    },
    [retrieveChildren],
  );

  const handleCheck = useCallback(
    (key: string) => {
      checkedState.toggle(key, nodes);
      const wasChecked = checkedState.checked.has(key);
      if (wasChecked) return;
      // Toggling ON — fetch unloaded children so they can be checked via reconcile
      const node = Tree.findNode({ tree: nodes, key });
      if (node == null) return;
      const toFetch = Tree.getDescendants(node).filter(
        (n) => n.children != null && n.children.length === 0,
      );
      for (const n of toFetch) fetchNodeChildren(n.key);
    },
    [checkedState, nodes, fetchNodeChildren],
  );

  const sort = useCallback(
    (a: Tree.Node, b: Tree.Node) => {
      if (isSection(a.key) || isSection(b.key)) return 0;
      const [aRes, bRes] = resourceStore.get([a.key, b.key]);
      if (aRes == null && bRes == null) return 0;
      if (aRes == null) return 1;
      if (bRes == null) return -1;
      if (aRes.id.type === "group" && bRes.id.type !== "group") return -1;
      if (aRes.id.type !== "group" && bRes.id.type === "group") return 1;
      return aRes.name.localeCompare(bRes.name);
    },
    [resourceStore],
  );

  const [selected, setSelected] = useState<string[]>([]);

  const treeProps = Tree.use({
    nodes,
    onExpand: handleExpand,
    selected,
    onSelectedChange: setSelected,
    sort,
  });

  const subscribe = useCallback(
    (callback: () => void, key: string) => resourceStore.onSet(callback, key),
    [resourceStore],
  );

  const renderItem = Component.renderProp(
    (itemProps: Tree.ItemProps<string>) => (
      <CheckboxItem
        {...itemProps}
        nodes={nodes}
        checkedState={checkedState}
        services={services}
        loadingRef={loadingRef}
        loadingListeners={loadingListenersRef}
        channelCount={channelCount}
        onCheck={handleCheck}
      />
    ),
  );

  return (
    <Flex.Box y style={{ height: "100%" }}>
      <Flex.Box y grow style={{ padding: "1rem", overflow: "auto" }}>
        <Tree.Tree<string, ontology.Resource>
          {...treeProps}
          showRules
          subscribe={subscribe}
          getItem={resourceStore.get.bind(resourceStore)}
        >
          {renderItem}
        </Tree.Tree>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            disabled={checkedState.checked.size === 0 || client == null || cluster == null}
            trigger={Triggers.SAVE}
            onClick={() => {
              if (client == null || cluster == null) return;
              const request = buildExportRequest(checkedState.checked);
              handleError(
                () => downloadSyc({ client, cluster, request, addStatus }),
                "Failed to export",
              );
            }}
          >
            Export
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
