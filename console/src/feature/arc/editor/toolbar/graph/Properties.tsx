// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/toolbar/graph/Properties.css";

import { arc } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Status } from "@synnaxlabs/lyra/status";
import { Arc, Diagram } from "@synnaxlabs/pluto";
import { box, type direction, location, xy } from "@synnaxlabs/x";
import { type FC, memo, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

export const Properties = memo((): ReactElement | null => {
  const selected = Session.Arc.useSelectSelected();
  const nodes = Arc.useNodes({ keys: selected });
  if (selected.length === 0 || nodes.length === 0)
    return (
      <Status.Summary center variant="disabled" hideIcon>
        Select an Arc element to configure its properties
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
  const config = Arc.useNodeConfig({ nodeKey });
  if (config == null) return null;
  const C = Arc.Graph.Node.REGISTRY[config.type];
  if (C == null) return null;
  return <NodeForm nodeKey={nodeKey} TypeForm={C.Form} />;
};

interface NodeFormProps extends IndividualConfigProps {
  TypeForm: FC;
}

const NodeForm = ({ nodeKey, TypeForm }: NodeFormProps): ReactElement => {
  const { form } = Arc.useNodeForm({ nodeKey });
  return (
    <Flex.Box className={CSS.BE("arc-properties", "individual")} y>
      <Form.Form<typeof Arc.Graph.Node.configZ> {...form}>
        <TypeForm />
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
  const handleError = Status.useErrorHandler();

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
            const orientation = el.className.match(/react-flow__handle-(\w+)/)?.[1];
            if (orientation == null || !location.isOuter(orientation))
              throw new Error("Cannot find the handle orientation.");
            return new Diagram.HandleLayout(dist, orientation);
          });
          return new Diagram.NodeLayout(node.key, nodeBox, handles);
        } catch (e) {
          handleError(e, "Failed to align nodes");
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
    <Flex.Box align="start" x className={CSS.BE("arc-properties", "multi")}>
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
