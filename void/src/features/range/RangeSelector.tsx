import { Header, Space } from "@synnaxlabs/pluto";
import { AiFillBoxPlot, AiOutlinePlus } from "react-icons/ai";

function RangeSelector() {
  return (
    <Space>
      <Header
        level="p"
        icon={<AiFillBoxPlot />}
        actions={[
          {
            children: <AiOutlinePlus />,
          },
        ]}
      >
        Range Selection
      </Header>
    </Space>
  );
}

const RangeSelectorItem = {
  key: "rangeSelector",
  icon: <AiFillBoxPlot />,
  content: <RangeSelector />,
};

export default RangeSelectorItem;
