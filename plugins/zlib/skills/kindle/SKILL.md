---
name: kindle
description: Send a local ebook to Kindle with the zlib CLI, configure its SMTP delivery, or diagnose a failed send. Use the bundled zlib skill to search or download a book first.
license: MIT
---

# Kindle delivery

Use `zlib kindle send` for a local file. It uses SMTP and does not need a
Z-Library login. If a book must be found or downloaded first, use the bundled
[zlib skill](../zlib/SKILL.md).

## Send

1. Confirm `zlib version` succeeds and the requested file exists. Resolve an
   ambiguous file choice before sending. Reuse the user's existing choice.
2. If the user requested delivery and settings are already configured, run the
   command below without a TTY. Let the CLI resolve `ZLIB_SMTP_PWD` from its
   environment and `.env` files; checking only the shell variable is insufficient.
3. If settings are missing or the user requests setup, read
   [SMTP setup](references/setup.md), complete the missing settings, then send.

```sh
zlib kindle send "./books/book.epub"
```

Pass the actual file path. Bare `zlib kindle` and `zlib kindle send` open terminal
prompts. Sending mail requires the user's request; an existing instruction to send
this file is sufficient, so do not ask for another confirmation.

Report the CLI result and file name. SMTP success means the sending server
accepted the message, not that the Kindle received it. After a timeout or uncertain
result, check the failure before retrying to avoid duplicate messages.

## Failures

| Result | Next action |
| --- | --- |
| Missing configuration or SMTP password | Follow [setup](references/setup.md); never request a password in chat |
| SMTP authentication rejected | Check sender account, provider authentication requirements and settings precedence |
| SMTP timeout / connection failure | Check the host, port and network; `ZLIB_PROXY` affects HTTP requests, not SMTP |
| Unsupported attachment | Report the CLI rejection; do not rename an extension to bypass validation |
| Attachment too large | Report the size error; encoding adds overhead to the SMTP message |
| Send succeeded, book missing | Check recipient, Amazon's approved sender list and any delivery response email |

The current CLI validates file types and size before sending. It rejects MOBI.
Its SMTP transport supports STARTTLS when offered, not implicit TLS on port 465;
use a provider endpoint compatible with that transport. Do not guess SMTP settings
from the email domain or promise that switching ports will fix an error.
