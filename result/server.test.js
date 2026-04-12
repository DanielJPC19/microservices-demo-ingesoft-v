const { collectVotesFromResult } = require('./server');

describe('collectVotesFromResult', () => {
  test('returns zeroes when result set is empty', () => {
    const result = { rows: [] };
    expect(collectVotesFromResult(result)).toEqual({ a: 0, b: 0 });
  });

  test('counts votes for option a', () => {
    const result = { rows: [{ vote: 'a', count: '7' }] };
    expect(collectVotesFromResult(result)).toEqual({ a: 7, b: 0 });
  });

  test('counts votes for option b', () => {
    const result = { rows: [{ vote: 'b', count: '3' }] };
    expect(collectVotesFromResult(result)).toEqual({ a: 0, b: 3 });
  });

  test('counts both options when both present', () => {
    const result = {
      rows: [
        { vote: 'a', count: '12' },
        { vote: 'b', count: '5' },
      ],
    };
    expect(collectVotesFromResult(result)).toEqual({ a: 12, b: 5 });
  });

  test('parses count strings as integers', () => {
    const result = { rows: [{ vote: 'a', count: '100' }] };
    const votes = collectVotesFromResult(result);
    expect(typeof votes.a).toBe('number');
    expect(votes.a).toBe(100);
  });
});
