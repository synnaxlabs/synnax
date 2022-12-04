import { LayoutRendererProps } from "@/features/layout";
import { TimeStamp } from "@synnaxlabs/client";
import { Space, Input, Header, Nav, Button } from "@synnaxlabs/pluto";
import { useForm } from "react-hook-form";
import { AiFillBoxPlot } from "react-icons/ai";
import { useDispatch } from "react-redux";
import { addRange } from "../store/slice";

const timeStringToNanoSeconds = (time: string) => {
  var p = time.split(":"),
    s = 0,
    m = 1;

  while (p.length > 0) {
    s += m * parseInt(p.pop() as string, 10);
    m *= 60;
  }

  return s * 1000000000;
};

const dateStringToNanoSeconds = (date: string) => {
  const dateObj = new Date(date);
  return dateObj.getTime() * 1000000;
};

export const DefineRange = ({ layoutKey, onClose }: LayoutRendererProps) => {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm();
  const dispatch = useDispatch();

  const onSubmit = (data: any) => {
    let start = dateStringToNanoSeconds(data.dateStart);
    start += timeStringToNanoSeconds(data.timeStart);
    let end = dateStringToNanoSeconds(data.dateEnd);
    end += timeStringToNanoSeconds(data.timeEnd);
    dispatch(
      addRange({
        name: data.name,
        key: data.name,
        start,
        end,
      })
    );
    onClose();
  };

  return (
    <Space grow>
      <Header level="h4" icon={<AiFillBoxPlot />}>
        Define a Range
      </Header>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleSubmit(onSubmit)(e);
        }}
        id="define-range"
      >
        <Space grow style={{ padding: "1rem" }} size="small">
          <Input.Item
            label="Name"
            helpText={errors.name?.message?.toString()}
            {...register("name")}
          />
          <Space direction="horizontal">
            <Input.Item
              label="Start Date"
              size="medium"
              helpText={errors.dateStart?.message?.toString()}
              {...register("dateStart", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Date}
            </Input.Item>
            <Input.Item
              label="Start Time"
              size="medium"
              helpText={errors.timeStart?.message?.toString()}
              {...register("timeStart", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Time}
            </Input.Item>
          </Space>
          <Space direction="horizontal">
            <Input.Item
              label="End Date"
              size="medium"
              helpText={errors.dateEnd?.message?.toString()}
              {...register("dateEnd", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Date}
            </Input.Item>
            <Input.Item
              label="End Time"
              size="medium"
              helpText={errors.timeEnd?.message?.toString()}
              {...register("timeEnd", { required: true })}
              style={{ flexGrow: "1" }}
            >
              {Input.Time}
            </Input.Item>
          </Space>
        </Space>
      </form>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button type="submit" form="define-range">
            Save
          </Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Space>
  );
};
