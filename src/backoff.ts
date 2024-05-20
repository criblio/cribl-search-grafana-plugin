const MAX_BACKOFF = 1000;

/**
 * A function that returns the next appropriate delay (ms) to use when backing off
 */
export type GetBackoffFunction = () => number;

/**
 * Creates a function that returns an exponential backoff delay. On each subsequent call, the function applies
 * some logic to calculate the next value for the delay. The logic works like this: take the initial number,
 * and after passing the next threshold, increase it. After reaching the last iteration threshold, the function
 * no longer increases the number. The value can never exceed MAX_BACKOFF.
 * @param initial The initial value to wait.
 * @returns a function that returns the next backoff delay
 */
export function createGetBackoff(initial = 200): GetBackoffFunction {
  let iteration = 0;
  const getBackoff = (): number => {
    const result = Math.min(Math.pow(initial, iterationPower(iteration)), MAX_BACKOFF);
    iteration++;
    return result;
  };
  return getBackoff;
}

/**
 * Some hardcoded thresholds before increasing. Backoff delay starts at the initial value, then increases
 * at each threshold -- e.g. it increases at the 10th iteration, again at the 15th iteration, etc.
 */
const THRESHOLDS = [10, 15, 17] as const;

function iterationPower(iteration: number): number {
  let index = 0;
  for (index = 0; index < THRESHOLDS.length; index++) {
    const threshold = THRESHOLDS[index];
    if (iteration < threshold) {
      return index + 1;
    }
  }
  return index + 1;
}
