import test from 'ava';

import { newClient } from '../../setupspecs';
import { DataType, Rate, TimeSpan, TimeStamp } from '../telem';
import { randomTypedArray } from '../util/telem';

const client = newClient();

const newChannel = async () => {
  return await client.channel.create({
    name: 'test',
    nodeId: 1,
    rate: Rate.Hz(1),
    dataType: DataType.Float64,
  });
};

test('TypedWriter - basic write', async (t) => {
  const ch = await newChannel();
  const writer = await client.data.newWriter(0, [ch.key]);
  try {
    await writer.write({ [ch.key]: randomTypedArray(10, ch.dataType) });
    await writer.commit();
  } finally {
    await writer.close();
  }
  t.true(true);
});

test('Client - basic write', async (t) => {
  const ch = await newChannel();

  const data = randomTypedArray(10, ch.dataType);

  await client.data.write(ch.key, TimeStamp.Seconds(1), data);

  await client.data.read(ch.key, TimeSpan.Zero, TimeSpan.Seconds(10000000));

  t.true(data.length === 10);
});
