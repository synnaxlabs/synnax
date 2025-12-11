import { Button, Flex, Form, Icon } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { useCallback } from "react";
import type z from "zod";

import { State } from "@/timeline/state";
import { type types } from "@/timeline/types";

export interface StatesProps {
  path: string;
}

export const States = ({ path }: StatesProps) => {
  const states = Form.useFieldList<string, z.infer<typeof types.stateZ>>(path);
  const handleAddState = useCallback(() => {
    states.push({
      key: id.create(),
      name: "State 1",
      nodes: [],
    });
  }, [states]);
  return (
    <Flex.Box y pack>
      {states.data.map((state) => (
        <State.State
          key={state}
          itemKey={state}
          path={`${path}.${state}`}
          onDelete={states.remove}
        />
      ))}
      <Button.Button
        variant="text"
        contrast={2}
        full="y"
        style={{ width: "calc(100% - 6px)", left: "3px", opacity: "0.7" }}
        onClick={handleAddState}
      >
        <Icon.Add />
      </Button.Button>
    </Flex.Box>
  );
};
