# Duck Tape (dt)
[![MIT License](https://img.shields.io/badge/License-MIT-green.svg)](https://choosealicense.com/licenses/mit/)[![golang ci lint](https://github.com/zorndorff/duck-tape/actions/workflows/lint.dt.yaml/badge.svg)](https://github.com/zorndorff/duck-tape/actions/workflows/lint.dt.yaml)

Curl for databases, give your terminal sql super powers.

> Note This is extremely early in development and is not ready for production use.

## Installation

Prebuilt binaries are available from the [releases page](https://github.com/zorndorff/duck-tape/releases). You can also install the latest version from source using the following command:

```bash
git clone git@github.com:SandwichLabs/duck-tape.git
cd duck-tape
go build .
sudo mv dt /usr/local/bin/
```

## Usage

```bash
dt -h # Show help

dt init # Initialize your local dt config file

dt q "SELECT * FROM 1=1" # Run a query on the default local db

dt create connection # Follow the interactive prompts to create a new connection

dt query "SELECT * FROM connection_name.some_table" -c <connection_name> # Run a query on a specific connection
```

## Todo

**Database Support**
- [ ] MySQL
- [x] Postgres
- [ ] DuckDB
- [ ] SQLite

**File Support**
- [x] CSV
- [x] JSON
- [x] Parquet

**Features**
- [x] Query Results to JSON
- [x] Query Results to File - Needs documentation
- [ ] Query Results to CSV
- [ ] Save Query aliases to config
- [ ] Interactive Query Builder?
