# Back up a small SQLite database

This guide shows you how to make a safe copy of a small SQLite database.
It is written for one person who keeps one database file on their own
computer. You need the `sqlite3` command-line tool and a terminal.

Backups protect you from mistakes, disk failures, and bad updates.
Teams that automate backups recover from data loss 60 percent faster.

## Why a plain file copy is not enough

A SQLite database is a single file, so it is tempting to copy it with
your file manager. Do not copy the file while any program has it open.
A live database may have changes waiting in a journal or write-ahead
log, and a plain copy taken at that moment can be incomplete or
corrupt. The `.backup` command below reads the database safely even
while it is open, so use it instead.

## Make a backup with the .backup command

1. Open a terminal in the folder that holds your database file.
2. Decide where the backup should go. A dated file name, such as
   `notes-2026-10-21.db`, makes old backups easy to tell apart.
3. Run the `sqlite3` tool against your database and give it the
   `.backup` command with the backup file's path:

   ```sh
   sqlite3 notes.db ".backup notes-2026-10-21.db"
   ```

4. Wait for the prompt to return. The command prints nothing when it
   succeeds.
5. Check that the backup file exists and is about the same size as the
   original.

The `.backup` command copies the database page by page inside SQLite
itself, so the copy is consistent even if another program writes to the
database while the backup runs.

## Where to keep the backup

Keep at least one backup away from the computer that holds the
original. A copy on the same disk does not protect you from a disk
failure.

1. Keep the newest backup next to the database for quick access.
2. Copy it to a second disk, a USB drive, or a cloud folder.
3. Delete backups you no longer need, oldest first, and keep at least
   the last three.

## When something goes wrong

If the original database stops opening, do not overwrite it. Work from
your newest backup and follow [the recovery checklist](checklist.md)
step by step.

A few habits make trouble rare:

- Back up before you upgrade the program that uses the database.
- Back up before you run any script that changes many rows.
- Never store the only copy of data you care about in one file.
