import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { NotificationSpec } from "@/status/Aggregator";
import { Circle } from "@/status/Circle";
import { Text } from "@/text";
import { Icon } from "@synnaxlabs/media";
import { toArray } from "@synnaxlabs/x";
import { ReactElement, isValidElement } from "react";

import "@/status/Notification.css";

export interface NotificationProps extends Align.SpaceProps {
  status: NotificationSpec;
  silence: (key: string) => void;
  actions?: ReactElement | Button.ButtonProps[];
}

export const Notification = ({
  status: { key, time, count, message, description, variant },
  silence,
  actions,
  className,
  ...props
}: NotificationProps): ReactElement => {
  return (
    <Align.Space
      className={CSS(CSS.B("notification"), className)}
      direction="y"
      key={time.toString()}
      empty
      {...props}
    >
      <Align.Space direction="x" justify="spaceBetween" grow style={{ width: "100%" }}>
        <Align.Space direction="x" align="center" size="small">
          <Circle style={{ height: "2.25rem", width: "2.5rem" }} variant={variant} />
          <Text.Text level="small" shade={7}>
            {`x${count}`}
          </Text.Text>
          <Text.DateTime
            className={CSS(CSS.BE("notification", "time"))}
            level="small"
            format="time"
          >
            {time}
          </Text.DateTime>
        </Align.Space>
        <Button.Icon
          className={CSS(CSS.BE("notification", "silence"))}
          variant="outlined"
          size="small"
          onClick={() => silence(key)}
        >
          <Icon.Close />
        </Button.Icon>
      </Align.Space>
      <Align.Space
        direction="y"
        align="start"
        className={CSS(CSS.BE("notification", "content"))}
        size="small"
      >
        <Text.Text
          className={CSS(CSS.BE("notification", "message"))}
          level="p"
          style={{ flexGrow: 1 }}
        >
          {message}
        </Text.Text>
        {description != null && (
          <Text.Text
            className={CSS(CSS.BE("notification", "description"))}
            level="small"
            style={{ flexGrow: 1 }}
          >
            {description}
          </Text.Text>
        )}
      </Align.Space>
      {actions != null && (
        <Align.Space
          direction="x"
          align="center"
          justify="end"
          className={CSS(CSS.BE("notification", "actions"))}
        >
          {toArray<ReactElement | Button.ButtonProps>(actions).map((a, i) => (
            <Action key={a.key} action={a} />
          ))}
        </Align.Space>
      )}
    </Align.Space>
  );
};

interface ActionProps {
  action: ReactElement | Button.ButtonProps;
}

const Action = ({ action }: ActionProps): ReactElement => {
  if (!isValidElement(action)) {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-type-assertion
    const props = action as Button.ButtonProps;
    return <Button.Button {...props} />;
  }
  return action;
};
