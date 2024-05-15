import { prependCriblOperator } from "utils";

describe('utils', () => {
  describe('prependCriblOperator', () => {
    [
      {
        query: 'print 42',
        expected: 'print 42',
      },
      {
        query: 'search in(foo) * | where bar="hello"',
        expected: 'search in(foo) * | where bar="hello"',
      },
      {
        query: 'find in(foo) * | where bar="hello"',
        expected: 'find in(foo) * | where bar="hello"',
      },
      {
        query: 'externaldata ["blah"] | summarize by foo',
        expected: 'externaldata ["blah"] | summarize by foo',
      },
      {
        query: 'needle',
        expected: 'cribl needle',
      },
      {
        query: '"needle"',
        expected: 'cribl "needle"',
      },
      {
        query: '* | where foo=42',
        expected: 'cribl * | where foo=42',
      },
      {
        query: 'dataset="$vt_dummy" event < 10',
        expected: 'cribl dataset="$vt_dummy" event < 10',
      },
      {
        query: 'cribl dataset="$vt_dummy" event < 10',
        expected: 'cribl dataset="$vt_dummy" event < 10',
      },
      {
        query: 'set logger_level="debug"; dataset="$vt_dummy" event < 10',
        expected: 'set logger_level="debug"; cribl dataset="$vt_dummy" event < 10',
      },
      {
        query: 'set logger_level="debug"; cribl dataset="$vt_dummy" event < 10',
        expected: 'set logger_level="debug"; cribl dataset="$vt_dummy" event < 10',
      },
      {
        query: 'set logger_level="debug"; set foo=42; dataset="$vt_dummy" event < 10',
        expected: 'set logger_level="debug"; set foo=42; cribl dataset="$vt_dummy" event < 10',
      },
      {
        query: 'let stage1 = foo; let stage2 = bar; set baz="biff"; dataset="$vt_dummy" event < 10',
        expected: 'let stage1 = foo; let stage2 = bar; set baz="biff"; cribl dataset="$vt_dummy" event < 10',
      },
      {
        query: 'let stage1 = foo; let stage2 = cribl bar; set baz="biff"; dataset="$vt_dummy" event < 10',
        expected: 'let stage1 = foo; let stage2 = cribl bar; set baz="biff"; cribl dataset="$vt_dummy" event < 10',
      },
      // These should never be run in this context, but at least we don't prepend cribl
      {
        query: '.show all queries',
        expected: '.show all queries',
      },
      {
        query: '.show objects(cribl_search_sample)',
        expected: '.show objects(cribl_search_sample)',
      },
    ].forEach(({ query, expected }) => {
      it(`should prepend cribl as needed: ${query}`, () => {
        expect(prependCriblOperator(query)).toBe(expected);
      });
    });
  });
});
