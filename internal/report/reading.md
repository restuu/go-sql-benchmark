## How to read the numbers

Every library does the same small database jobs (add a row, read a row, and so on). For each job we measure three things, and lower is better for all three:

- **p95 (latency)**: how long one call takes, counting the slow ones. 95 out of 100 calls were at least this fast, so it shows what a user occasionally waits, which an average would hide.
- **B/op (memory)**: bytes of memory the program asked for on each call ("op" is one call). More memory means more work later for Go's garbage collector, the part that frees memory.
- **CPU ns/op (processor time)**: nanoseconds of processor time spent on each call. One nanosecond is a billionth of a second.

Each number is compared with plain `database/sql`, Go's built-in way to talk to a SQL database, which is the baseline. That is what the `x` figures mean: `1.00x` is the same as the baseline, `1.04x` is 4% more, `0.50x` is half.

So "p95 1.04x, B/op 1.43x, CPU ns/op 1.03x" reads: this library's calls take 4% longer, use 43% more memory and use 3% more processor time than `database/sql`.

A library's **score** combines its three ratios over every scored case into one number (see the [glossary](#glossary) for how). `1.00` is the same as `database/sql`, `2.00` is twice as costly, `0.50` half. Lower is better. Other terms, such as scenario, load profile and ORM, are in the [glossary](#glossary).
