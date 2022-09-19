import { ReactElement } from "react";
import { Space } from "@synnaxlabs/pluto";

type SplitSlideProps = {
  widths?: (string | number)[];
  children: ReactElement[];
  justify?: "start" | "center" | "end" | "between" | "around" | "evenly";
  align?: "start" | "center" | "end" | "stretch";
};

export default function SplitSlide({
  widths = [],
  children = [],
  justify = "center",
  align = "center",
}: SplitSlideProps) {
  return (
    <Space direction="horizontal" grow justify={justify} align={align}>
      {children.map((child, index) => (
        <div
          style={{
            flexGrow: index < widths.length ? 0 : 1,
            width: index < widths.length ? widths[index] : 0,
          }}
        >
          {child}
        </div>
      ))}
    </Space>
  );
}
