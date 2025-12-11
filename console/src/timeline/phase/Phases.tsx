import { Button, Form, Icon, Timeline } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { useCallback } from "react";
import type z from "zod";

import { Phase } from "@/timeline/phase/Phase";
import { type types } from "@/timeline/types";

export const Phases = () => {
  const phases = Form.useFieldList<string, z.infer<typeof types.phaseZ>>("phases");
  const handleAddPhase = useCallback(() => {
    phases.push({
      key: id.create(),
      name: "Phase 1",
      states: [],
    });
  }, [phases]);
  const handleDeletePhase = useCallback(
    (path: string) => {
      console.log("delete phase", path);
      phases.remove(path);
    },
    [phases],
  );
  return (
    <Timeline.Track full>
      {phases.data.map((phase, index) => (
        <Phase
          key={phase}
          path={`phases.${phase}`}
          index={index}
          onDelete={() => phases.remove(phase)}
        />
      ))}
      <Button.Button
        variant="text"
        contrast={2}
        full="y"
        style={{ height: "100%", opacity: "0.7" }}
        onClick={handleAddPhase}
      >
        <Icon.Add />
      </Button.Button>
    </Timeline.Track>
  );
};
