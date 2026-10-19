# Send to Kindle

[README](../README.md) · [Configuration](configuration.md)

zlib sends local files to your Kindle email address through your SMTP server.
Sending a local file does not require a Z-Library session.

## Set up delivery

Find your Kindle email address in Amazon's Personal Document Settings and add
your sender email to the approved sender list. Then run:

```sh
zlib kindle config
```

Enter the Kindle address, sender address, SMTP host and port. You can also edit
each value with `zlib config kindle` or `/config kindle` in the shell. The CLI
saves these settings in `~/.config/zlib/kindle.json`.

The port selects how zlib secures the connection. zlib sends the password only
after TLS is established, and fails if the server offers no TLS.

| Port | Connection | Example |
| --- | --- | --- |
| `465` or `994` | Implicit TLS: TLS starts as soon as zlib connects. Providers often call this SSL. | `smtp.163.com:465`, `smtp.qq.com:465` |
| Any other port, normally `587` | Plain connection, upgraded with STARTTLS | `smtp.gmail.com:587` |

Use the port your provider documents for the connection type you choose.

When you send interactively without `ZLIB_SMTP_PWD`, a masked prompt asks for the
SMTP password. Saving writes it to the [global `.env`
file](configuration.md#create-or-edit-the-global-file) and makes it available to
the current send. Use an app password if your mail provider requires one. For
non-interactive sends, set `ZLIB_SMTP_PWD` in that file beforehand.

## Send a file

```sh
zlib kindle send ./book.epub
```

`zlib kindle` and `zlib kindle send` open the same file picker as `/kindle`.
In a terminal, sending a path also asks for confirmation. The settings, password,
progress, error, and result pages are shared with the TUI; closing a standalone
page returns to the terminal. Non-interactive sends still run directly.

For a book
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
| SMTP password not set | Enter it in the TUI prompt, or set `ZLIB_SMTP_PWD` in the global `.env` for non-interactive use |
| SMTP authentication failed | Check the sender account and provider's app-password requirements |
| `smtp send failed: host:port (mode): …` | The error shows the server, the port and the TLS mode zlib used. A timeout or `EOF` usually means the port expects the other TLS mode: use `465` for SSL/TLS or `587` for STARTTLS. Otherwise, check the host and network access |
| Send succeeded but no book arrived | Check the Kindle address, approved sender list and Amazon's response email |

Use `zlib kindle --help` and `zlib kindle send --help` for command options.
