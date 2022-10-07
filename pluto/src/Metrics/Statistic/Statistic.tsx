import clsx from "clsx";
import Space from "../../Atoms/Space/Space";
import Text, { textLevel } from "../../Atoms/Typography/Text";
import "./Value.css";

export interface ValueProps {
  value: number;
  level?: textLevel;
  label?: string;
  variant?: "primary" | "error";
  color?: string;
}

export const Statistic = ({
  value,
  level = "h4",
  variant = "primary",
  label,
  color,
}: ValueProps) => {
  return (
    <Space empty direction="vertical" align="center" justify="center">
      <Text
        className={clsx(
          "pluto-value__text",
          variant && `pluto-value__text--${variant}`
        )}
        level={level}
      >
        {value}
      </Text>
      {label && (
        <Text className="pluto-value__label" level="small">
          {label}
        </Text>
      )}
    </Space>
  );
};

export default Statistic;
