## Glossary

Plain-language meanings of the terms used in this report. `REPORT.md` uses the same terms.

### What is measured

- **latency**: the time one call to the database takes, from asking until the answer is back.
- **percentile**: sort all the measured times from fastest to slowest and look at one position. **p50** is the middle one (half the calls were faster), **p95** is the time 95 out of 100 calls beat, and **p99** is the time 99 out of 100 beat, so it reflects the rare slow calls.
- **p50**: the median latency: half the calls were faster, half slower.
- **p95**: the main latency figure in this report. Unlike an average, a few slow calls show up in it.
- **p99**: see percentile. Stress-profile p99 on macOS also includes a constant 100 to 200 ms stall between the host and the Docker virtual machine, which is not the library's doing.
- **throughput** (**ops/s**): how many calls finish per second, added up over all workers. Higher is better; it is shown for context and is not part of the score.
- **B/op**: bytes of memory allocated per operation (one call).
- **allocs/op**: how many separate memory allocations one call makes. Fewer is usually cheaper.
- **CPU ns/op**: processor time per call, in nanoseconds (billionths of a second).
- **garbage collector**: the part of Go that frees memory the program no longer uses. The more a call allocates, the more work the garbage collector has to do, and that work is included in the CPU figure.
- **ratio**: a measurement divided by the same measurement for `database/sql` in the same case. `1.04x` means 4% more than the baseline; `0.50x` means half.
- **baseline**: `database/sql`, the reference every other library is compared against.
- **geometric mean**: a way of averaging ratios. Twice as slow in one case and twice as fast in another cancel out to 1.00, which an ordinary average would not give.
- **score**: for each library, the weighted geometric mean of its p95, B/op and CPU ns/op ratios over all the cases it was judged on (the weights are listed under Top 5). `1.00` equals `database/sql`; lower is better. It never depends on the other libraries in the run.
- **median**: the middle value. Each case is run several times and the middle result is reported, so one unlucky run does not decide it.
- **spread** (**p95 spread**): how far apart the p95 of the repeats of a case were, as a share of the median. A `*` next to a figure means the repeats disagreed by more than 10%, so read it as approximate.
- **p95 vs database/sql**: the library's p95 compared with `database/sql`'s p95 in the same table, as a percentage. `+20.0%` means 20% slower than `database/sql`, `-20.0%` means 20% faster.
- **error rate** (**err %**): the share of calls that failed, as a percentage. A `!` next to a library marks a case above 1%; it is shown but not ranked.
- **n/a**: the library cannot do that scenario with its own query features (only with hand-written SQL), so the case is left out of the ranking.

### What the benchmark does

- **scenario**: one kind of database job. There are six, listed next.
- **insert**: add one new row to a table.
- **update**: change every column of one existing row.
- **delete**: remove one row.
- **select**: read one row by its id.
- **join**: read the orders of one customer together with that customer's details, which live in two tables (10 rows).
- **union**: combine two lists, the customers and the suppliers of one city, into one result (40 rows).
- **load profile**: how hard the database is pushed. **low** is a quiet application: 4 callers, each pausing 10 ms between calls. **stress** is a busy one: 100 callers hitting the database without pausing.
- **low**: the quiet load profile (4 workers, 10 ms between calls).
- **stress**: the busy load profile (100 workers, no pauses).
- **worker**: one caller running calls one after another. More workers means more calls at the same moment.
- **warm-up**: the first seconds of each run, which are thrown away so start-up effects do not count.
- **measured window**: the seconds after the warm-up during which the numbers are recorded.
- **case**: one scenario under one load profile, written scenario/profile, for example join/low.
- **connection pool**: a set of ready database connections that callers share, so each call does not open its own.

### Kinds of library

- **database/sql**: Go's standard package for SQL databases. You write the SQL yourself.
- **driver**: the piece that speaks to the database. Every `database/sql`-based library here uses the same one (`pgx`), so only the library on top differs.
- **SQL-first**: you write plain SQL and the library helps run it and read the rows back.
- **query builder**: you build the query with Go code (functions and method calls) instead of writing SQL text.
- **ORM** (object-relational mapper): you work with Go structs and the library writes the SQL for you.
- **code generation**: a tool reads your database or SQL and writes Go code for you ahead of time (`make gen` regenerates it here).
- **PostgreSQL**: the database used for every measurement (version 15).
- **honorable mention**: a library that is measured and shown but not ranked against the others, because it only works with PostgreSQL (`pgx`, `pgx_pool`), so code written for it cannot move to another database.

### Qualitative terms (judgements, not measurements)

- **dynamic query**: a query whose shape changes at run time, for example a search that adds a filter only when the user fills in that field.
- **raw SQL**: SQL written as plain text inside the Go code. A library that needs it for something has no built-in way to do that.
- **type safety**: how many mistakes the Go compiler can catch before the program runs, such as a misspelled column name or a wrong type.
