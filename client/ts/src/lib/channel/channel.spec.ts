import test from 'ava';

import Synnax from '../client';
import { DataType, Rate } from '../telem';

const client = new Synnax({ host: 'localhost', port: 8080 });

test('Channel - create', async (t) => {
  const channel = await client.channel.create({
    name: 'test',
    nodeId: 1,
    rate: Rate.Hz(1),
    dataType: DataType.Float32,
  });
  t.is(channel.name, 'test');
  t.is(channel.nodeId, 1);
  t.deepEqual(channel.rate, Rate.Hz(1));
  t.deepEqual(channel.dataType, DataType.Float32);
});

test('Channel - retrieve by key', async (t) => {
  const channel = await client.channel.create({
    name: 'test',
    nodeId: 1,
    rate: Rate.Hz(1),
    dataType: DataType.Float32,
  });
  const retrieved = (await client.channel.retrieveByKeys(channel.key))[0];
  t.is(retrieved.name, 'test');
  t.is(retrieved.nodeId, 1);
  t.deepEqual(retrieved.rate, Rate.Hz(1));
  t.deepEqual(retrieved.dataType, DataType.Float32);
});
