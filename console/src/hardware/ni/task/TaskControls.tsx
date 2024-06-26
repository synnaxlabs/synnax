import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Status } from "@synnaxlabs/pluto";

export interface ControlsProps {
  onStartStop: () => void;
  startingOrStopping: boolean;
  onConfigure: () => void;
  configuring: boolean;
  state?: task.State<{ running?: boolean; message?: string }>;
}

export const Controls = ({
  state,
  onStartStop,
  startingOrStopping,
  onConfigure,
  configuring,
}: ControlsProps) => (
  <Align.Space
    direction="x"
    style={{
      borderRadius: "1rem",
      border: "var(--pluto-border)",
      padding: "2rem",
    }}
    justify="spaceBetween"
  >
    <Align.Space direction="x">
      {state?.details?.message != null && (
        <Status.Text variant={state?.variant as Status.Variant}>
          {state?.details?.message}
        </Status.Text>
      )}
    </Align.Space>
    <Align.Space direction="x">
      <Button.Icon
        loading={startingOrStopping}
        disabled={startingOrStopping || state == null}
        onClick={onStartStop}
        variant="outlined"
      >
        {state?.details?.running === true ? <Icon.Pause /> : <Icon.Play />}
      </Button.Icon>
      <Button.Button loading={configuring} disabled={configuring} onClick={onConfigure}>
        Configure
      </Button.Button>
    </Align.Space>
  </Align.Space>
);
