export type ChannelProps = {
  key?: string;
  name?: string;
  rate?: number;
  node_id?: number;
  data_type?: string;
  density?: number;
};

export default class Channel {
  key: string;
  name: string;
  node_id: number;
  data_rate: number;
  data_type: string;
  density: number;

  constructor({
    key = '',
    name = '',
    rate = 0,
    node_id = 0,
    data_type = '',
    density = 0,
  }: ChannelProps) {
    this.key = key;
    this.name = name;
    this.node_id = node_id;
    this.data_rate = rate;
    this.data_type = data_type;
    this.density = density;
  }
}
