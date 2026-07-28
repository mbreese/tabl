# tabl

## Utilities for viewing / working with tab-delimited text files.

This is going to be a whole port over from an earlier Python package: [tabutils](https://github.com/mbreese/tabutils). At the moment, only the view and less functions are working.

For more information, see here: [https://compgen.io/tabl](https://compgen.io/tabl).

Note: The `tabl less` pager forces 256-color output to keep formatting/colors consistent on terminals like tmux/screen (especially on RHEL8).

## Parquet

`view`, `less`, and `parquet2tab` can read Parquet files. Parquet is detected
automatically from the file's magic bytes, so no flag is needed:

    tabl view data.parquet
    tabl less data.parquet
    tabl parquet2tab data.parquet > data.txt

Use `--parquet` to force it for a file that isn't named/detected as one.

The parquet schema supplies the header. Nested structs are flattened into dotted
column names (`addr.city`), while lists and maps are rendered as JSON in a single
cell. NULL cells are written as an empty value; use `--na=STRING` to change that
(e.g. `--na=NA`).

Because a parquet file's schema lives in a footer at the end of the file,
reading one requires random access -- parquet cannot be read from a pipe, so a
real file path is required.

## Examples

![Demo](https://github.com/compgen-io/tabl-docs/raw/master/assets/img/tabl-demo-2.gif)
