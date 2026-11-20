import { useMemo } from "react";
import { sortedRowOrder, type CsvSort } from "../../lib/csv";

// A semantic table of literal cell text, sorted only in the display (FS-03.R120).
export function CsvTable({ rows, columns, sort, onSort }: {
  rows: string[][];
  columns: number;
  sort: CsvSort | null;
  onSort: (sort: CsvSort) => void;
}) {
  const order = useMemo(() => sortedRowOrder(rows, sort), [rows, sort]);
  const choose = (column: number) => onSort({
    column,
    direction: sort?.column === column && sort.direction === "ascending" ? "descending" : "ascending",
  });
  return (
    <div className="file-viewer-csv" tabIndex={0} role="region" aria-label="CSV table">
      <table className="file-viewer-csv-table">
        <thead>
          <tr>
            {Array.from({ length: columns }, (_, column) => (
              <th key={column} scope="col" aria-sort={sort?.column === column ? sort.direction : undefined}>
                <button type="button" onClick={() => choose(column)}>
                  {column + 1}
                  {sort?.column === column && <span aria-hidden="true">{sort.direction === "ascending" ? " ▲" : " ▼"}</span>}
                </button>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {order.map((index) => (
            <tr key={index}>
              {rows[index].map((cell, column) => <td key={column}>{cell}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
