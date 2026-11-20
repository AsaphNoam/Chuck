// Bounded CSV parsing and sorting for the shared file viewer (FS-03.R120, TS-08.R174).
// Cells are literal text: nothing here interprets HTML or evaluates formulas.

export const CSV_MAX_ROWS = 1000;
export const CSV_MAX_COLUMNS = 100;
export const CSV_MAX_CELLS = 20000;

export type CsvParse =
  | { ok: true; rows: string[][]; columns: number }
  | { ok: false; reason: "malformed" | "too-large" };

export const isCsvPath = (path: string) => /\.csv$/i.test(path);

export function parseCsv(source: string): CsvParse {
  const text = source.charCodeAt(0) === 0xfeff ? source.slice(1) : source;
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let quoted = false; // inside a quoted field
  let started = false; // current field began with a quote or has characters
  let cells = 0;
  const endRecord = (): CsvParse | null => {
    row.push(cell);
    rows.push(row);
    cells += row.length;
    if (rows.length > CSV_MAX_ROWS || row.length > CSV_MAX_COLUMNS || cells > CSV_MAX_CELLS) return { ok: false, reason: "too-large" };
    if (row.length !== rows[0].length) return { ok: false, reason: "malformed" };
    row = []; cell = ""; started = false;
    return null;
  };
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (quoted) {
      if (ch === '"') {
        if (text[i + 1] === '"') { cell += '"'; i++; }
        else {
          quoted = false;
          const next = text[i + 1];
          if (next !== undefined && next !== "," && next !== "\n" && next !== "\r") return { ok: false, reason: "malformed" };
        }
      } else cell += ch;
    } else if (ch === '"') {
      if (started) return { ok: false, reason: "malformed" };
      quoted = true; started = true;
    } else if (ch === ",") {
      row.push(cell); cell = ""; started = false;
      if (row.length > CSV_MAX_COLUMNS) return { ok: false, reason: "too-large" };
    } else if (ch === "\n" || ch === "\r") {
      if (ch === "\r" && text[i + 1] === "\n") i++;
      const failed = endRecord();
      if (failed) return failed;
    } else { cell += ch; started = true; }
  }
  if (quoted) return { ok: false, reason: "malformed" };
  // A trailing final newline does not create an extra empty record.
  if (started || cell !== "" || row.length > 0) {
    const failed = endRecord();
    if (failed) return failed;
  }
  if (rows.length === 0) return { ok: false, reason: "malformed" };
  return { ok: true, rows, columns: rows[0].length };
}

export type CsvSort = { column: number; direction: "ascending" | "descending" };

const DECIMAL = /^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/;
const parseNumber = (value: string) => {
  const trimmed = value.trim();
  if (!DECIMAL.test(trimmed)) return null;
  const n = Number(trimmed);
  return Number.isFinite(n) ? n : null;
};

/** At least one nonblank value, and every nonblank value a finite decimal. */
export function isNumericColumn(rows: string[][], column: number): boolean {
  let seen = false;
  for (const row of rows) {
    if (row[column].trim() === "") continue;
    if (parseNumber(row[column]) === null) return false;
    seen = true;
  }
  return seen;
}

const collator = new Intl.Collator(undefined);

/** Original row indices in display order. Blanks last and ties in file order, in both directions. */
export function sortedRowOrder(rows: string[][], sort: CsvSort | null): number[] {
  const order = rows.map((_, index) => index);
  if (!sort) return order;
  const { column } = sort;
  const sign = sort.direction === "ascending" ? 1 : -1;
  const numeric = isNumericColumn(rows, column);
  const blank = (index: number) => rows[index][column].trim() === "";
  return order.sort((a, b) => {
    const blankA = blank(a), blankB = blank(b);
    if (blankA || blankB) return blankA === blankB ? a - b : blankA ? 1 : -1;
    const compared = numeric
      ? parseNumber(rows[a][column])! - parseNumber(rows[b][column])!
      : collator.compare(rows[a][column], rows[b][column]);
    return compared === 0 ? a - b : sign * compared;
  });
}
