import { List } from "@synnaxlabs/pluto";

import { RenderableLayout, useSelectLayouts } from "@/features/layout";

export const LayoutList = (): JSX.Element => {
  const layouts = useSelectLayouts().filter(
    (layout) => layout.type === "visualization"
  );
  return (
    <List<RenderableLayout> data={layouts}>
      <List.Column.Header<RenderableLayout>
        columns={[
          {
            key: "title",
            label: "Title",
          },
        ]}
      />
      <List.Core.Virtual itemHeight={30} style={{ height: "100%" }}>
        {List.Column.Item}
      </List.Core.Virtual>
    </List>
  );
};
