/**
 * Cribl Search backend requires a fully-formed query including the "cribl" operator.
 * Here we're applying best effort to auto-prepending the "cribl" operator where it needs to be.
 * This is far from 100% foolproof, but covers 99% of the happy paths.
 * @param query the query as the user entered it
 * @returns the query with "cribl" operator prepended as needed, if we were able to
 */
export function prependCriblOperator(query: string): string {
  // This tries to capture the first "word" of the root query statement, along with any preceding
  // "set" or "let" statements, and everything that comes after the first word.  NOTE: This doesn't
  // touch any "let" statements.  We might enhance that in the future, but for now, users need to
  // put their own "cribl" operator in those stage statements.
  const firstWordRegex = /^((?:\s*(?:set|let)\s+[^;]+;)*\s*)((\w+|['"*]).*)$/;
  const matches = query.trim().match(firstWordRegex);
  if (matches != null) {
    const firstWord = matches[3];
    // We recognize certain operators that don't need "cribl" prepended
    if (!['cribl', 'externaldata', 'find', 'print', 'search'].includes(firstWord)) {
      return `${matches[1]}cribl ${matches[2]}`;
    }
  }
  return query;
}
