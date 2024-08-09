If you use GoLand, you should move .run directory to rosetta-cli/.idea/runConfigurations. `mv .run /path/to/rosetta-cli/.idea/runConfigurations`. And please edit the directory path that fits your running environment.

For testing testnet in automated way, we should fill account infos for
`construction.prefunded_accounts` field info at `config.json` file.

We need at least 2 accounts which have enough KAIA to test transfer scenario.

Tip for testing.
- check:data
  - Give a value to `end_conditions.index` of `config.json` to limit the work scope.
  - If you don't write `end_conditions.index` it would run endlessly because there is no end conditions.
  - Or you can set a `end_conditions.reconciliation_coverage.coverage`.
    - When coverage exceeds value, check:data process will exit.
