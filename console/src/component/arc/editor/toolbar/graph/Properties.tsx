// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import {
  Arc,
  Button,
  Diagram,
  Flex,
  Form,
  Icon,
  Input,
  Status,
} from "@synnaxlabs/pluto";
import { box, deep, type direction, location, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useMemo } from "react";

import { Session } from "@/session";

export const Properties = memo((): ReactElement | null => {
  const selected = Session.Arc.useSelectSelected();
  const nodes = Arc.useSelectNodes({ keys: selected });
  if (selected.length === 0 || nodes.length === 0)
    return (
      <Status.Summary center variant="disabled" hideIcon>
        Select an Arc element to configure its properties.
      </Status.Summary>
    );
  if (selected.length > 1) return <MultiConfig nodes={nodes} />;
  return <IndividualConfig key={nodes[0].key} nodeKey={nodes[0].key} />;
});
Properties.displayName = "PropertiesControls";

interface IndividualConfigProps {
  nodeKey: string;
}

const IndividualConfig = ({ nodeKey }: IndividualConfigProps): ReactElement | null => {
  const config = Arc.useSelectNodeConfig({ nodeKey });
  const initialValues = useMemo(() => deep.copy(config), [config]);
  const dispatch = Arc.useSingleDispatch();
  const formMethods = Form.use<typeof Arc.Graph.Node.configZ>({
    schema: Arc.Graph.Node.configZ,
    values: initialValues,
    sync: true,
    onChange: useCallback(
      ({ values: config }: Form.OnChangeArgs<typeof Arc.Graph.Node.configZ>) =>
        dispatch(arc.setNodeConfig({ key: nodeKey, config: deep.copy(config) })),
      [dispatch, nodeKey],
    ),
  });
  if (config == null) return null;
  const C = Arc.Graph.Node.REGISTRY[config.type];
  if (C == null) return null;
  return (
    <Flex.Box style={{ height: "100%", padding: "2rem" }} y>
      <Form.Form<typeof Arc.Graph.Node.configZ> {...formMethods}>
        <C.Form {...formMethods} key={nodeKey} />
      </Form.Form>
    </Flex.Box>
  );
};

interface MultiElementPropertiesProps {
  nodes: Diagram.Node[];
}

const MultiConfig = ({ nodes }: MultiElementPropertiesProps): ReactElement => {
  const dispatch = Arc.useSingleDispatch();
  const viewport = Session.Arc.useSelectViewport();

  const getLayouts = () =>
    nodes
      .map((node) => {
        try {
          const nodeEl = Diagram.selectNode(node.key);
          const nodeBox = box.construct(node.position, box.dims(box.construct(nodeEl)));
          const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
          const nodeElBox = box.construct(nodeEl);
          const handles = Array.from(handleEls).map((el) => {
            const pos = box.center(box.construct(el));
            const dist = xy.scale(
              xy.translation(box.topLeft(nodeElBox), pos),
              1 / viewport.zoom,
            );
            const match = el.className.match(/react-flow__handle-(\w+)/);
            if (match == null)
              throw new Error(`[arc] - cannot find handle orientation`);
            const orientation = location.construct(match[1]) as location.Outer;
            return new Diagram.HandleLayout(dist, orientation);
          });
          return new Diagram.NodeLayout(node.key, nodeBox, handles);
        } catch (e) {
          console.error(e);
        }
        return null;
      })
      .filter((el) => el !== null);

  const align = (direction: direction.Direction): void => {
    const newPositions = Diagram.alignNodesAlongDirection(getLayouts(), direction);
    dispatch(
      newPositions.map(({ key, box: b }) =>
        arc.setNodePosition({ key, position: box.topLeft(b) }),
      ),
    );
  };

  return (
    <Flex.Box align="start" x style={{ padding: "2rem" }}>
      <Input.Item label="Align">
        <Flex.Box x>
          <Button.Button tooltip="Align nodes vertically" onClick={() => align("x")}>
            <Icon.Align.YCenter />
          </Button.Button>
          <Button.Button tooltip="Align nodes horizontally" onClick={() => align("y")}>
            <Icon.Align.XCenter />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
    </Flex.Box>
  );
};
