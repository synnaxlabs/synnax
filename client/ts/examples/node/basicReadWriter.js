import { DataType, Synnax, TimeSpan, TimeStamp } from "@synnaxlabs/client";

const client = new Synnax({
    host: "localhost",
    port: 9090,
    username: "synnax",
    password: "seldon",
    secure: false
});



const time_channel = await client.channels.create({
    name: "basic_read_write_time",
    isIndex: true,
    dataType: DataType.TIMESTAMP
});

const data_channel = await client.channels.create({
    name: "basic_read_write_data",
    isIndex: false,
    dataType: DataType.FLOAT32,
    index: time_channel.key,
});

const N_SAMPLES = 500;

const start = TimeStamp.now();


const data = Float32Array.from({ length: N_SAMPLES }, (_, i) => Math.sin(i / 100));
const time = BigInt64Array.from({ length: N_SAMPLES }, (_, i) => start.add(TimeSpan.milliseconds(i)).bigInt);

await client.telem.write("basic_read_write_time", start, time);
// console.log("HERE")
// // await data_channel.write(start, data);

console.log("HERE")

client.close();