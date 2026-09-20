# Third-party components

- Cobra and pflag: BSD-3-Clause; mousetrap: Apache-2.0. Versions are in go.mod/go.sum; module distributions include their licenses.
- Poppler runs as a separate executable and includes GPL-licensed components. The Debian image retains package copyright/license files under /usr/share/doc/. Corresponding package source is available through Debian.
- poppler-data mapping licenses are recorded in its Debian copyright file; Noto CJK fonts use the SIL Open Font License.
- ReportLab is used only by the development fixture generator, not linked into the Go executable. See its upstream BSD license.

This file identifies dependencies; it does not relicense them or replace upstream license terms.
