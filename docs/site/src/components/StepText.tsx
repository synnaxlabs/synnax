import { Icon } from "@synnaxlabs/media";
import { Text } from "@synnaxlabs/pluto/text";

export interface StepTextProps extends Text.TextProps {
  step: string | number;
}

export const StepText = ({ step, children, ...props }: StepTextProps) => (
  <Text.Text {...props}>
    <span
      style={{
        color: "var(--pluto-gray-l7)",
        display: "inline-flex",
        alignItems: "center",
      }}
    >
      Step {step} <Icon.Arrow.Right style={{ margin: "0 1rem" }} />
    </span>
    {children}
  </Text.Text>
);
