import { getRanges } from './metricTimeSplit';

describe('querySplit', () => {
  // FIXME add more tests, for cases like too small chunk-sizes etc.
  it('should split time range into chunks', () => {
    const start = Date.parse('2022-02-06T14:10:03');
    const end = Date.parse('2022-02-06T14:11:03');
    const step = 10 * 1000;

    expect(getRanges(start, end, step, 35000)).toStrictEqual([
      [Date.parse('2022-02-06T14:10:00'), Date.parse('2022-02-06T14:10:10')],
      [Date.parse('2022-02-06T14:10:20'), Date.parse('2022-02-06T14:10:40')],
      [Date.parse('2022-02-06T14:10:50'), Date.parse('2022-02-06T14:11:10')],
    ]);
  });
});
