# Send to Kindle

[README](../README.md) · [Configuration](configuration.md)

zlib sends local files to your Kindle email address through your SMTP server.
Sending a local file does not require a Z-Library session.

## Set up delivery

Find your Kindle email address in Amazon's Personal Document Settings and add
your sender email to the approved sender list. Then run:

```sh
zlib kindle
```

Enter the Kindle address, sender address, SMTP host and port. You can also edit
each value with `zlib config kindle` or `/config kindle` in the shell. The CLI
saves these settings in `~/.config/zlib/kindle.json`.

Open the [global `.env` file](configuration.md#create-or-edit-the-global-file) in
your editor and add `ZLIB_SMTP_PWD` with the password required by your mail
provider. If the provider requires an app password, use that instead of your
account password. The CLI reads this value at send time.

## Send a file

```sh
zlib kindle send ./book.epub
```

Run `zlib kindle send` without a path to select a file interactively. For a book
you are downloading, copy its full reference from the search results and use:

```sh
zlib download "id:hash" --send-to-kindle
```

Replace `id:hash` with the selected result. SMTP success confirms that the sender's
mail server accepted the message; delivery to the device depends on Amazon's
sender and document checks.

## If delivery fails

| Symptom | Check |
| --- | --- |
| SMTP password not set | Set `ZLIB_SMTP_PWD` in the global `.env` used by this user |
| SMTP authentication failed | Check the sender account and provider's app-password requirements |
| SMTP connection failed | Check the configured host, port and network access |
| Send succeeded but no book arrived | Check the Kindle address, approved sender list and Amazon's response email |

Use `zlib kindle --help` and `zlib kindle send --help` for command options.
