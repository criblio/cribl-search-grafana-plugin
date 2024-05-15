import { createGetBackoff } from "backoff";

describe('getBackoff', () => {
  [
    { n: 0, p: 1 },
    { n: 9, p: 1 },
    { n: 10, p: 2 },
    { n: 14, p: 2 },
    { n: 15, p: 3 },
    { n: 16, p: 3 },
    { n: 17, p: 4 },
    { n: 50, p: 4 },
  ].forEach((element) => {
    it(`should set to the power of ${element.p} for ${element.n} iterations`, async () => {
      const getBackoff = createGetBackoff(2);
      let backoff = getBackoff();
      for (let index = 0; index < element.n; index++) {
        backoff = getBackoff();
      }
      expect(backoff).toBe(Math.pow(2, element.p));
    });
  });
});
