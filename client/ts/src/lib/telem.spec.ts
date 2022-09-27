import test from 'ava';

import {
  DataType,
  Density,
  Rate,
  Size,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from './telem';

// |||||| TimeStamp ||||||

test('TimeStamp - construct', (t) => {
  const ts = new TimeStamp(1000);
  t.true(ts.equals(TimeSpan.Microseconds()));
});

test('TimeStamp - span', (t) => {
  const ts = new TimeStamp(0);
  t.true(ts.span(new TimeStamp(1000)).equals(TimeSpan.Microseconds()));
});

test('TimeStamp - range', (t) => {
  const ts = new TimeStamp(0);
  t.true(
    ts
      .range(new TimeStamp(1000))
      .equals(new TimeRange(ts, TimeSpan.Microseconds()))
  );
});

test('TimeStamp - spanRange', (t) => {
  const ts = new TimeStamp(0);
  t.true(
    ts
      .spanRange(TimeSpan.Microseconds())
      .equals(new TimeRange(ts, ts.add(TimeSpan.Microseconds())))
  );
});

test('TimeStamp - isZero', (t) => {
  const ts = new TimeStamp(0);
  t.true(ts.isZero());
});

test('TimeStamp - after', (t) => {
  const ts = new TimeStamp(0);
  t.true(ts.after(new TimeStamp(-1)));
  const ts2 = new TimeStamp(1);
  t.false(ts2.after(new TimeStamp(1)));
});

test('TimeStamp - before', (t) => {
  const ts = new TimeStamp(0);
  t.true(ts.before(new TimeStamp(1)));
  const ts2 = new TimeStamp(1);
  t.false(ts2.before(new TimeStamp(1)));
});

test('TimeStamp - beforeEq', (t) => {
  const ts = new TimeStamp(0);
  t.true(ts.beforeEq(new TimeStamp(1)));
  const ts2 = new TimeStamp(1);
  t.true(ts2.beforeEq(new TimeStamp(1)));
  const ts3 = new TimeStamp(2);
  t.false(ts3.beforeEq(new TimeStamp(1)));
});

test('TimeStamp - afterEq', (t) => {
  const ts = new TimeStamp(0);
  t.true(ts.afterEq(new TimeStamp(-1)));
  const ts2 = new TimeStamp(1);
  t.true(ts2.afterEq(new TimeStamp(1)));
  const ts3 = new TimeStamp(0);
  t.false(ts3.afterEq(new TimeStamp(1)));
});

test('TimeStamp - add', (t) => {
  const ts = new TimeStamp(0);
  t.true(
    ts
      .add(TimeSpan.Microseconds())
      .equals(new TimeStamp(TimeSpan.Microseconds(1)))
  );
});

test('TimeStamp - sub', (t) => {
  const ts = new TimeStamp(TimeSpan.Microseconds());
  t.true(ts.sub(TimeSpan.Microseconds()).equals(new TimeStamp(0)));
});

// |||||| TimeSpan ||||||

test('TimeSpan - construct from static', (t) => {
  t.true(TimeSpan.Nanoseconds(1).equals(1));
  t.true(TimeSpan.Microseconds(1).equals(1000));
  t.true(TimeSpan.Milliseconds(1).equals(1000000));
  t.true(TimeSpan.Seconds(1).equals(1e9));
  t.true(TimeSpan.Minutes(1).equals(6e10));
  t.true(TimeSpan.Hours(1).equals(36e11));
});

test('TimeSpan - seconds', (t) => {
  t.is(TimeSpan.Seconds(1).seconds(), 1);
});

test('TimeSpan - isZero', (t) => {
  t.true(TimeSpan.Zero.isZero());
  t.false(TimeSpan.Seconds(1).isZero());
});

test('TimeSpan - add', (t) => {
  t.true(TimeSpan.Seconds(1).add(TimeSpan.Second).equals(2e9));
});

test('TimeSpan - sub', (t) => {
  t.true(TimeSpan.Seconds(1).sub(TimeSpan.Second).isZero());
});

// |||||| Rate ||||||

test('Rate - construct', (t) => {
  t.true(new Rate(1).equals(1));
});

test('Rate - period', (t) => {
  t.true(new Rate(1).period().equals(TimeSpan.Second));
});

test('Rate - sampleCount', (t) => {
  t.true(new Rate(1).sampleCount(TimeSpan.Second) == 1);
});

test('Rate - byteCount', (t) => {
  t.true(new Rate(1).byteCount(TimeSpan.Second, Density.Bit64) == 8);
});

test('Rate - span', (t) => {
  t.true(new Rate(1).span(4).equals(TimeSpan.Seconds(4)));
});

test('Rate - byteSpan', (t) => {
  t.true(
    new Rate(1)
      .byteSpan(new Size(32), Density.Bit64)
      .equals(TimeSpan.Seconds(4))
  );
});

test('Rate - Hz', (t) => {
  t.true(Rate.Hz(1).equals(1));
});

test('Rate - KHz', (t) => {
  t.true(Rate.KHz(1).equals(1e3));
});

// |||||| TimeRange ||||||

test('TimeRange - construct', (t) => {
  const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
  t.true(tr.start.equals(new TimeStamp(0)));
  t.true(tr.end.equals(new TimeStamp(1000)));
});

test('TimeRange - span', (t) => {
  const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
  t.true(tr.span().equals(TimeSpan.Microsecond));
});

test('TimeRange - isValid', (t) => {
  const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
  t.true(tr.isValid());
  const tr2 = new TimeRange(new TimeStamp(1000), new TimeStamp(0));
  t.false(tr2.isValid());
});

test('TimeRange - isZero', (t) => {
  const tr = new TimeRange(new TimeStamp(0), new TimeStamp(0));
  t.true(tr.isZero());
  const tr2 = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
  t.false(tr2.isZero());
});

test('TimeRange - swap', (t) => {
  const tr = new TimeRange(new TimeStamp(0), new TimeStamp(1000));
  t.true(
    tr.swap().equals(new TimeRange(new TimeStamp(1000), new TimeStamp(0)))
  );
});

// |||||| DATA TYPE ||||||

test('DataType - json serialization', (t) => {
  const dt = DataType.Int32;
  const v = JSON.parse(JSON.stringify({ dt }));
  t.true(v.dt === 'int32');
});
