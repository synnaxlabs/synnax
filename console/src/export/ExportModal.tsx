// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, ontology, type Synnax as Client } from "@synnaxlabs/client";
import {
  Button,
  Channel,
  Component,
  Flex,
  Flux,
  Icon,
  Input,
  List,
  Nav,
  type Ontology,
  Status,
  Synnax,
  Text,
  Tree,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo, useState } from "react";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { IndeterminateCheckbox } from "@/components/IndeterminateCheckbox";
import { type BackupExportRequest, downloadBackup } from "@/export/download";
import { type CheckedState, useCheckedState } from "@/export/useCheckedState";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { type Service } from "@/ontology/service";
import { useServices } from "@/ontology/ServicesProvider";
import { Triggers } from "@/triggers";

export const EXPORT_LAYOUT_TYPE = "exportSynnax";

export const EXPORT_LAYOUT: Layout.BaseState = {
  key: EXPORT_LAYOUT_TYPE,
  type: EXPORT_LAYOUT_TYPE,
  name: "Manage Core.Export Synnax",
  icon: "Export",
  location: "modal",
  window: { resizable: true, size: { height: 600, width: 500 }, navTop: true },
};

interface SectionConfig {
  key: string;
  label: string;
  icon: ReactElement;
  /** If set, find resources of this type directly under root instead of a named group */
  rootType?: ontology.ResourceType;
}

const SECTIONS: SectionConfig[] = [
  { key: "section:workspaces", label: "Workspaces", icon: <Icon.Workspace /> },
  { key: "section:users", label: "Users", icon: <Icon.User /> },
  { key: "section:devices", label: "Devices", icon: <Icon.Device /> },
  { key: "section:tasks", label: "Tasks", icon: <Icon.Task /> },
  { key: "section:ranges", label: "Ranges", icon: <Icon.Range />, rootType: "range" },
  { key: "section:channels", label: "Channels", icon: <Icon.Channel /> },
];

const SELECT_ALL_KEY = "select-all";
const SECTION_MAP = new Map(SECTIONS.map((s) => [s.key, s]));
const isSection = (key: string): boolean => key.startsWith("section:");
const CHANNELS_SECTION_KEY = "section:channels";

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


interface CheckboxItemProps extends Tree.ItemProps<string> {
  nodes: Tree.Node[];
  checkedState: CheckedState;
  services: ReturnType<typeof useServices>;
  channelCount: number;
}

const CheckboxItem = ({
  nodes,
  checkedState,
  services,
  channelCount,
  ...props
}: CheckboxItemProps): ReactElement | null => {
  const { itemKey } = props;
  const { isChecked, isIndeterminate, toggle, checked } = checkedState;
  const itemChecked = isChecked(itemKey);
  const indeterminate = isIndeterminate(itemKey, nodes);
  const handleToggle = useCallback(
    () => toggle(itemKey, nodes),
    [toggle, itemKey, nodes],
  );

  if (itemKey === SELECT_ALL_KEY)
    return (
      <Tree.Item {...props}>
        <IndeterminateCheckbox
          checked={itemChecked}
          indeterminate={indeterminate}
          onChange={handleToggle}
        />
        <Text.Text level="small" weight={500} style={{ userSelect: "none" }}>
          Synnax
        </Text.Text>
      </Tree.Item>
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
        (n) =>
          n.key !== itemKey &&
          !n.key.startsWith("group:") &&
          checked.has(n.key),
      ).length;
    }
    return (
      <Tree.Item {...props}>
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
    <Tree.Item {...props}>
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

type ArrayFields = "workspace_keys" | "user_keys" | "device_keys" | "task_keys" | "range_keys" | "channel_keys";

const TYPE_TO_FIELD: Partial<Record<string, ArrayFields>> = {
  workspace: "workspace_keys",
  user: "user_keys",
  device: "device_keys",
  task: "task_keys",
  range: "range_keys",
  channel: "channel_keys",
};

const NUMERIC_FIELDS = new Set<string>(["task_keys", "channel_keys"]);

const buildExportRequest = (checked: Set<string>): BackupExportRequest => {
  const request: BackupExportRequest = {};
  for (const key of checked) {
    if (key === SELECT_ALL_KEY || key.startsWith("section:") || key.startsWith("group:"))
      continue;
    const id = ontology.idZ.safeParse(key);
    if (!id.success) continue;
    const field = TYPE_TO_FIELD[id.data.type];
    if (field == null) continue;
    const arr = (request[field] ??= []) as Array<string | number>;
    arr.push(NUMERIC_FIELDS.has(field) ? Number(id.data.key) : id.data.key);
  }
  return request;
};

/** Recursively fetch all children of an ontology ID and build tree nodes. */
const fetchTreeRecursive = async (
  client: Client,
  parentID: ontology.ID,
  resourceStore: Flux.UnaryStore<string, ontology.Resource>,
  services: Record<string, Service>,
): Promise<Tree.Node[]> => {
  const children = await client.ontology.retrieveChildren(parentID);
  const filtered = children.filter((r) => {
    const svc = services[r.id.type];
    return svc.visible == null || svc.visible(r);
  });
  filtered.forEach((r) => resourceStore.set(r));

  const nodes: Tree.Node[] = [];
  for (const r of filtered) {
    const svc = services[r.id.type];
    let childNodes: Tree.Node[] | undefined;
    if (svc.hasChildren)
      childNodes = await fetchTreeRecursive(
        client,
        r.id,
        resourceStore,
        services,
      );
    nodes.push({
      key: ontology.idToString(r.id),
      children: childNodes,
    });
  }
  return nodes;
};

type ExportPage = "selection" | "channelSettings";

const PAGE_NAMES: Record<ExportPage, string> = {
  selection: "Manage Core.Export Synnax",
  channelSettings: "Manage Core.Channel Export Settings",
};

export const ExportModal = ({ layoutKey }: Layout.RendererProps): ReactElement => {
  const services = useServices();
  const resourceStore = Flux.useStore<Ontology.FluxSubStore>().resources;
  const client = Synnax.use();
  const cluster = Cluster.useSelect();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const checkedState = useCheckedState();
  const dispatch = useDispatch();
  const [page, setPage] = useState<ExportPage>("selection");

  const navigate = useCallback(
    (p: ExportPage) => {
      setPage(p);
      dispatch(Layout.rename({ key: layoutKey, name: PAGE_NAMES[p] }));
    },
    [dispatch, layoutKey],
  );

  const [exportData, setExportData] = useState(false);
  const [startTime, setStartTime] = useState(0);
  const [endTime, setEndTime] = useState(Date.now() * 1e6);

  const [nodes, setNodes] = useState<Tree.Node[]>([]);
  const [channelCount, setChannelCount] = useState(0);
  const [loading, setLoading] = useState(true);

  // Load the full tree on mount
  useAsyncEffect(async () => {
    if (client == null) return;
    try {
      const [rootChildren, channels] = await Promise.all([
        client.ontology.retrieveChildren(ontology.ROOT_ID),
        client.channels.retrieve({ internal: false }),
      ]);
      rootChildren.forEach((r) => resourceStore.set(r));
      setChannelCount(channels.length);

      const sectionNodes: Tree.Node[] = [];
      for (const section of SECTIONS) {
        let children: Tree.Node[] = [];
        if (section.rootType != null) {
          // Resources without a parent group (e.g., ranges) — query by type
          const items = await client.ontology.retrieve({
            types: [section.rootType],
          });
          items.forEach((r) => resourceStore.set(r));
          children = items.map((r) => ({
            key: ontology.idToString(r.id),
          }));
        } else {
          const group = rootChildren.find((r) => r.name === section.label);
          if (group != null)
            children = await fetchTreeRecursive(
              client,
              group.id,
              resourceStore,
              services,
            );
        }
        sectionNodes.push({ key: section.key, children });
      }
      setNodes([{ key: SELECT_ALL_KEY, children: sectionNodes }]);
    } catch (e) {
      handleError(e);
    } finally {
      setLoading(false);
    }
  }, [client]);

  const hasChannelsSelected = useMemo(() => {
    for (const key of checkedState.checked) {
      if (
        key === SELECT_ALL_KEY ||
        key.startsWith("section:") ||
        key.startsWith("group:")
      )
        continue;
      const parsed = ontology.idZ.safeParse(key);
      if (parsed.success && parsed.data.type === "channel") return true;
    }
    return false;
  }, [checkedState.checked]);

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
    selected,
    onSelectedChange: setSelected,
    sort,
    initialExpanded: [SELECT_ALL_KEY],
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
        channelCount={channelCount}
      />
    ),
  );

  return (
    <Flex.Box y style={{ height: "100%", overflow: "hidden" }}>
      {page === "selection" && (
        <Flex.Box y grow style={{ padding: "1rem", overflow: "auto", minHeight: 0 }}>
          {loading ? (
            <Flex.Box y grow justify="center" align="center">
              <Icon.Loading style={{ fontSize: "2rem" }} />
              <Text.Text level="p" style={{ marginTop: "0.5rem" }}>
                Loading resources...
              </Text.Text>
            </Flex.Box>
          ) : (
            <Tree.Tree<string, ontology.Resource>
              {...treeProps}
              showRules
              subscribe={subscribe}
              getItem={resourceStore.get.bind(resourceStore)}
            >
              {renderItem}
            </Tree.Tree>
          )}
        </Flex.Box>
      )}
      {page === "channelSettings" && (
        <Flex.Box y grow style={{ padding: "1.5rem", overflow: "auto", minHeight: 0 }}>
          <Text.Text level="h4" weight={500}>
            Channel Export Settings
          </Text.Text>
          <Flex.Box
            x
            align="center"
            gap="small"
            style={{ marginTop: "1.5rem" }}
          >
            <Input.Switch value={exportData} onChange={setExportData} />
            <Text.Text level="p">Export time series data</Text.Text>
          </Flex.Box>
          <Flex.Box
            x
            align="center"
            style={{ gap: "1rem", marginTop: "1rem", paddingLeft: "1.5rem" }}
          >
            <Flex.Box y style={{ gap: "0.25rem" }}>
              <Text.Text
                level="small"
                style={{
                  color: exportData
                    ? "var(--pluto-gray-l9)"
                    : "var(--pluto-gray-l5)",
                }}
              >
                Start
              </Text.Text>
              <Input.DateTime
                value={startTime}
                onChange={setStartTime}
                onlyChangeOnBlur
                disabled={!exportData}
                level="small"
                variant="outlined"
              />
            </Flex.Box>
            <Icon.Arrow.Right
              style={{
                marginTop: "1rem",
                color: exportData
                  ? "var(--pluto-gray-l9)"
                  : "var(--pluto-gray-l5)",
              }}
            />
            <Flex.Box y style={{ gap: "0.25rem" }}>
              <Text.Text
                level="small"
                style={{
                  color: exportData
                    ? "var(--pluto-gray-l9)"
                    : "var(--pluto-gray-l5)",
                }}
              >
                End
              </Text.Text>
              <Input.DateTime
                value={endTime}
                onChange={setEndTime}
                onlyChangeOnBlur
                disabled={!exportData}
                level="small"
                variant="outlined"
              />
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      )}
      <Modals.BottomNavBar>
        <Nav.Bar.End>
          {page === "selection" && hasChannelsSelected && (
            <Button.Button
              variant="outlined"
              onClick={() => navigate("channelSettings")}
            >
              Next
            </Button.Button>
          )}
          {page === "channelSettings" && (
            <Button.Button
              variant="outlined"
              onClick={() => navigate("selection")}
            >
              Back
            </Button.Button>
          )}
          <Button.Button
            variant="filled"
            disabled={
              checkedState.checked.size === 0 ||
              client == null ||
              cluster == null ||
              (page === "selection" && hasChannelsSelected)
            }
            trigger={Triggers.SAVE}
            onClick={() => {
              if (client == null || cluster == null) return;
              const request = buildExportRequest(checkedState.checked);
              if (exportData) {
                request.include_data = true;
                request.time_range = { start: startTime, end: endTime };
              }
              handleError(
                () => downloadBackup({ client, cluster, request, addStatus }),
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
