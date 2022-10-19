import { Header, Space } from "@synnaxlabs/pluto";
import { AiFillBoxPlot, AiOutlinePlus } from "react-icons/ai";

function RangeSelector() {
  return (
    <Space>
      <Header
        level="p"
        text="Range Selector"
        icon={<AiFillBoxPlot />}
        actions={[
          {
            icon: <AiOutlinePlus />,
          },
        ]}
      />
    </Space>
  );
}

const RangeSelectorItem = {
  key: "rangeSelector",
  icon: <AiFillBoxPlot />,
  content: <RangeSelector />,
};

export default RangeSelectorItem;
