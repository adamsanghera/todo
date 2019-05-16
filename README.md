# todo

This is basically a wrapper around vim, with some interesting logic for parsing files to create memories and reminders.

# Roadmap

1. support `/remember X` to specify how many consecutive days something should appear in memories
1. support `/reminder X` to say remind me in "X" days.

# Notes on achieving the roadmap

The application needs to know how far into the past to look.

Otherwise, I can create a centralized memory system that stores these lines, and instead read only from this centralized system.

It might as well be a sqlite3 db file:

``` { .sql }
CREATE TABLE reminder (
    created  DATE,
    content  TEXT,
    appear   DATE,
    order    NUMERIC,

    PRIMARY KEY (created, order)
);

CREATE TABLE memory (
    created    DATE,
    disappear  DATE,
    content    TEXT,

    PRIMARY KEY (created, order)
);
```

Calling `todo` will scan files in the root folder, and insert memories and reminders into the database, then pop scanned files (that aren't today's) into the `archive` folder.

The directory structure will look like this:

```
.
|--- .archive
|    | ...              # archive of old journals, mined for reminders/remembers
|    `--- yesterday.md
|--- .db                # sqlite3 db
`--- today.md           # today's journal
```

After memories are printed into the "today" file, they can be deleted. This cleaning house allow for future schema flexibility. In the future when the schema is stable, we can just keep them around.
