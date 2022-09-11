import StreamReader from './StreamReader';
import { Sample } from './sample';

export default class SamplePool {
  state: Map<string, Float32Array>;
  reader: StreamReader;

  async subscribe(...series: SamplePoolSeries[]) {
    await this.reader.subscribe(...series.map((s) => s.channelKey));
    series.forEach((s) => {
      this.state[s.channelKey] = new Float32Array(s.retention);
    });
  }

  async unsubscribe(...channelKeys: string[]) {
    await this.reader.unsubscribe(...channelKeys);
    channelKeys.forEach((key) => {
      delete this.state[key];
    });
  }

  async read() {
    const samples = await this.reader.read();
    samples.forEach((sample) => {
      const series = this.state[sample.channelKey];
      series.shift();
      series.push(sample.value);
    });
  }
}

class SamplePoolSeries {
  channelKey: string;
  retention: number;
  data: Float32Array;

  constructor(channelKey: string, retention: number) {
    this.channelKey = channelKey;
    this.retention = retention;
    this.data = new Float32Array(retention);
  }

  push(samples: Sample[]) {
    samples.forEach((sample) => {
      this.shift();
      this.data.set(sample.value, this.data.length - 1);
    });
  }

  shift() {
    shiftLeftBy(this.data, 2);
  }
}

const shiftLeftBy = (arr: Float32Array, n: number) => {
  for (let i = 0; i < arr.length; i++) {
    arr[i] = arr[i + n];
  }
};
